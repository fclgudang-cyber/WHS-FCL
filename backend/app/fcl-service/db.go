// Package main — koneksi & helper DB untuk fcl-service (port dari server/db.js / Pg.php).
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var (
	db     *sql.DB
	colMu  sync.RWMutex
	colCache = map[string]map[string]string{}

	tableNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

func openDB(dsn string) error {
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	return db.Ping()
}

func queryAll(query string, args ...any) ([]map[string]any, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := map[string]any{}
		for i, c := range cols {
			v := vals[i]
			// []byte → string untuk JSON agar tidak jadi base64
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[c] = v
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func queryOne(query string, args ...any) (map[string]any, error) {
	rows, err := queryAll(query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

type pgError struct {
	Status int
	Code   string
	Detail string
	Hint   string
	Msg    string
}

func (e *pgError) Error() string { return e.Msg }

func missingColumnError(col, table string) *pgError {
	return &pgError{Status: 400, Code: "PGRST204",
		Msg: fmt.Sprintf("Could not find the '%s' column of '%s' in the schema cache", col, table)}
}
func missingTableError(table string) *pgError {
	return &pgError{Status: 404, Code: "42P01", Msg: fmt.Sprintf(`relation "%s" does not exist`, table)}
}
func invalidTable(table string) *pgError {
	return &pgError{Status: 400, Msg: fmt.Sprintf("invalid table name: %s", table)}
}

func loadTableColumns(table string) (map[string]string, error) {
	rows, err := db.Query(`select column_name, data_type from information_schema.columns where table_schema = 'public' and table_name = $1`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]string{}
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}
		cols[name] = dataType
	}
	return cols, nil
}

// ensureFclTable membuat tabel FCL minimal bila benar-benar belum ada.
// Dependency lain (ent core-service) tidak pernah disentuh.
func ensureFclTable(table string) error {
	_, err := db.Exec(fmt.Sprintf(`create table if not exists %s (
		id bigserial primary key,
		"createdAt" text
	)`, quoteIdent(table)))
	return err
}

func getTableColumns(table string) (map[string]string, error) {
	if !tableNameRe.MatchString(table) {
		return nil, invalidTable(table)
	}
	colMu.RLock()
	if c, ok := colCache[table]; ok {
		colMu.RUnlock()
		return c, nil
	}
	colMu.RUnlock()

	colMu.Lock()
	defer colMu.Unlock()
	if c, ok := colCache[table]; ok {
		return c, nil
	}
	cols, err := loadTableColumns(table)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		// Self-healing: database produksi sering hanya punya tabel ent core-service.
		// Buat tabel FCL otomatis agar fitur tidak gagal dengan "relation does not exist".
		if err = ensureFclTable(table); err != nil {
			return nil, err
		}
		cols, err = loadTableColumns(table)
		if err != nil {
			return nil, err
		}
	}
	colCache[table] = cols
	return cols, nil
}

// addDynamicColumn menambahkan kolom yang dipakai frontend tetapi belum ada
// di database (misal field baru dari hasil kustomisasi).
func addDynamicColumn(table, col string) error {
	if !tableNameRe.MatchString(table) || !tableNameRe.MatchString(col) {
		return fmt.Errorf("invalid identifier: %s.%s", table, col)
	}
	_, err := db.Exec(fmt.Sprintf(`alter table %s add column if not exists %s text`, quoteIdent(table), quoteIdent(col)))
	if err != nil {
		return err
	}
	colMu.Lock()
	delete(colCache, table)
	colMu.Unlock()
	cols, err := loadTableColumns(table)
	if err != nil {
		return err
	}
	colMu.Lock()
	colCache[table] = cols
	colMu.Unlock()
	return nil
}

func resolveColumn(table, col string) (string, error) {
	colMu.RLock()
	cols, ok := colCache[table]
	colMu.RUnlock()
	if ok {
		if dt, found := cols[col]; found {
			return dt, nil
		}
	}
	if err := addDynamicColumn(table, col); err != nil {
		return "", err
	}
	colMu.RLock()
	cols = colCache[table]
	colMu.RUnlock()
	if dt, found := cols[col]; found {
		return dt, nil
	}
	return "", missingColumnError(col, table)
}

var filterOps = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true,
	"in": true, "ilike": true, "isnull": true,
}

func unquoteIdent(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1:len(s)-1]
		s = strings.ReplaceAll(s, `""`, `"`)
		return s
	}
	return s
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isInteger(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func coerceParam(dataType string, value string) any {
	switch categoryOf(dataType) {
	case "integer":
		if i, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
			return i
		}
		return value
	case "numeric":
		if f, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return f
		}
		return value
	default:
		return value
	}
}

func categoryOf(dataType string) string {
	switch dataType {
	case "integer", "smallint", "bigint":
		return "integer"
	case "numeric", "double precision", "real":
		return "numeric"
	case "boolean":
		return "boolean"
	case "date":
		return "date"
	case "timestamp with time zone":
		return "timestamptz"
	case "timestamp without time zone":
		return "timestamp"
	case "uuid":
		return "uuid"
	default:
		return "text"
	}
}

// buildWhere parses query params berprefix f_ → clause WHERE + args
// (offset: placeholder awal saat digabung dengan SET params).
func buildWhere(table string, query map[string][]string, startIndex int) (string, []any, error) {
	var clauses []string
	var params []any
	for key, values := range query {
		if !strings.HasPrefix(key, "f_") || len(values) == 0 {
			continue
		}
		rawValue := values[0]
		body := key[2:]
		col := body
		op := "eq"
		if sep := strings.LastIndex(body, "__"); sep > 0 && filterOps[body[sep+2:]] {
			col = body[:sep]
			op = body[sep+2:]
		}
		col = unquoteIdent(col)
		dataType, err := resolveColumn(table, col)
		if err != nil {
			return "", nil, err
		}
		quoted := quoteIdent(col)
		paramBase := fmt.Sprintf("$%d", startIndex+len(params)+1)
		switch op {
		case "eq", "neq":
			clauses = append(clauses, fmt.Sprintf("%s %s %s", quoted, map[bool]string{true: "=", false: "<>"}[op == "eq"], paramBase))
			params = append(params, coerceParam(dataType, rawValue))
		case "gt", "gte", "lt", "lte":
			sym := map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<="}[op]
			clauses = append(clauses, fmt.Sprintf("%s %s %s", quoted, sym, paramBase))
			params = append(params, coerceParam(dataType, rawValue))
		case "in":
			parts := []string{}
			for _, p := range strings.Split(rawValue, ",") {
				if s := strings.TrimSpace(p); s != "" {
					parts = append(parts, s)
				}
			}
			if len(parts) == 0 {
				clauses = append(clauses, "false")
			} else {
				placeholders := []string{}
				for _, p := range parts {
					params = append(params, coerceParam(dataType, p))
					placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+len(params)))
				}
				clauses = append(clauses, fmt.Sprintf("%s in (%s)", quoted, strings.Join(placeholders, ", ")))
			}
		case "ilike":
			clauses = append(clauses, fmt.Sprintf("%s ilike %s", quoted, paramBase))
			params = append(params, rawValue)
		case "isnull":
			clauses = append(clauses, fmt.Sprintf("%s is %snull", quoted, map[bool]string{true: "", false: "not "}[rawValue == "true"]))
		}
	}
	if len(clauses) == 0 {
		return "", nil, nil
	}
	return " where " + strings.Join(clauses, " and "), params, nil
}

func buildOrderBy(order string) string {
	if strings.TrimSpace(order) == "" {
		return ""
	}
	var keys []string
	for _, part := range strings.Split(order, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		col := part
		dir := "asc"
		lower := strings.ToLower(part)
		if strings.HasSuffix(lower, ".desc") {
			col = part[:len(part)-len(".desc")]
			dir = "desc"
		} else if strings.HasSuffix(lower, ".asc") {
			col = part[:len(part)-len(".asc")]
		}
		col = unquoteIdent(col)
		if col == "" {
			continue
		}
		suffix := map[string]string{"asc": " asc", "desc": " desc nulls last"}[dir]
		keys = append(keys, quoteIdent(col)+suffix)
	}
	if len(keys) == 0 {
		return ""
	}
	return " order by " + strings.Join(keys, ", ")
}

func prepareProjection(table string, selectParam string) (string, error) {
	if strings.TrimSpace(selectParam) == "" || selectParam == "*" {
		return "*", nil
	}
	list := []string{}
	for _, s := range strings.Split(selectParam, ",") {
		if t := unquoteIdent(strings.TrimSpace(s)); t != "" {
			list = append(list, t)
		}
	}
	for _, col := range list {
		if col == "*" {
			continue
		}
		if _, err := resolveColumn(table, col); err != nil {
			return "", err
		}
	}
	quoted := []string{}
	for _, col := range list {
		quoted = append(quoted, quoteIdent(col))
	}
	return strings.Join(quoted, ", "), nil
}

// prepareRow: validasi kolom + coerce nilai insert/update
func prepareRow(table string, row map[string]any) (map[string]any, error) {
	cols, err := getTableColumns(table)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for key, value := range row {
		dataType, ok := cols[key]
		if !ok {
			if err := addDynamicColumn(table, key); err != nil {
				return nil, err
			}
			colMu.RLock()
			cols, ok = colCache[table]
			colMu.RUnlock()
			if !ok || cols[key] == "" {
				return nil, missingColumnError(key, table)
			}
			dataType = cols[key]
		}
		out[key] = coerceValue(dataType, value)
	}
	return out, nil
}
func coerceValue(dataType string, value any) any {
	if value == nil {
		return nil
	}
	switch categoryOf(dataType) {
	case "integer":
		switch v := value.(type) {
		case bool:
			return b2i(v)
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		case string:
			if isInteger(v) {
				n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
				return n
			}
			return v
		}
		// JSON decode default = float64
		if f, ok := value.(float64); ok {
			return int64(f)
		}
		return value
	case "numeric":
		switch v := value.(type) {
		case bool:
			if v {
				return 1
			}
			return 0
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return f
			}
			return v
		case float64, int, int64:
			if f, ok := toFloat(value); ok {
				return f
			}
			return value
		}
		return value
	case "boolean":
		switch v := value.(type) {
		case bool:
			return v
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			switch s {
			case "true", "t", "1", "yes", "ya", "on":
				return true
			case "false", "f", "0", "no", "tidak", "off":
				return false
			}
			return v != ""
		case float64:
			return v != 0
		}
		return value
	default:
		switch v := value.(type) {
		case bool:
			if v {
				return "true"
			}
			return "false"
		case float64, int, int64:
			if f, ok := toFloat(v); ok {
				return strconv.FormatFloat(f, 'f', -1, 64)
			}
			return fmt.Sprint(v)
		case map[string]any, []any:
			b, _ := json.Marshal(v)
			return string(b)
		default:
			return fmt.Sprint(value)
		}
	}
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// castRows: convert nilai hasil SELECT agar JSON keluar seperti PostgREST
func castRows(table string, rows []map[string]any) ([]map[string]any, error) {
	cols, err := getTableColumns(table)
	if err != nil {
		return rows, nil
	}
	for _, row := range rows {
		for key, v := range row {
			dtype, ok := cols[key]
			if !ok {
				continue
			}
			switch categoryOf(dtype) {
			case "integer":
				switch n := v.(type) {
				case int64:
					row[key] = n
				case float64:
					row[key] = int64(n)
				case string:
					if i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64); err == nil {
						row[key] = i
					}
				}
			case "numeric":
				switch n := v.(type) {
				case float64:
					row[key] = n
				case int64:
					row[key] = float64(n)
				case string:
					if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
						row[key] = f
					}
				}
			case "boolean":
				switch n := v.(type) {
				case bool:
					row[key] = n
				case string:
					s := strings.ToLower(strings.TrimSpace(n))
					row[key] = s == "t" || s == "true" || s == "1"
				}
			case "timestamptz":
				if ts, ok := v.(time.Time); ok {
					row[key] = ts.UTC().Format("2006-01-02T15:04:05.000000-07:00")
				} else if s, ok := v.(string); ok {
					// Normalisasi 'YYYY-MM-DD HH:MM:SS.mmm+TZ' → ISO
					row[key] = strings.Replace(s, " ", "T", 1)
				}
			case "timestamp":
				// biarkan apa adanya
			}
		}
	}
	return rows, nil
}
