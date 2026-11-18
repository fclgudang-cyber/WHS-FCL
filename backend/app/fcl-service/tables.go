// tables.go — CRUD generik tabel FCL (port dari server/routes/tables.js / TablesController.php).
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	if pe, ok := err.(*pgError); ok {
		writeJSON(w, pe.Status, map[string]any{
			"message": pe.Msg, "code": nullIfEmpty(pe.Code),
			"details": nullIfEmpty(pe.Detail), "hint": nullIfEmpty(pe.Hint),
		})
		return
	}
	writeJSON(w, 400, map[string]any{"message": err.Error(), "code": nil, "details": nil, "hint": nil})
}
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func tableName(r *http.Request) string {
	// path: /api/tables/{table}
	p := strings.TrimPrefix(r.URL.Path, "/api/tables/")
	p = strings.TrimSuffix(p, "/")
	// table tidak boleh mengandung '/'
	return strings.TrimSuffix(p, "/")
}

// Jangan pernah ekspos hash password dari tabel users
func stripUsersPassword(table string, rows []map[string]any) []map[string]any {
	if table != "users" {
		return rows
	}
	for _, r := range rows {
		delete(r, "password")
	}
	return rows
}

func handleList(w http.ResponseWriter, r *http.Request, table string) {
	cols, err := getTableColumns(table)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = cols

	q := r.URL.Query()
	projection, err := prepareProjection(table, q.Get("select"))
	if err != nil {
		writeError(w, err)
		return
	}
	where, params, err := buildWhere(table, q, 0)
	if err != nil {
		writeError(w, err)
		return
	}
	orderBy := buildOrderBy(q.Get("order"))
	countExact := q.Get("count") == "exact" && (q.Get("head") == "1" || q.Get("head") == "true")
	if countExact {
		qry := fmt.Sprintf("select count(*)::int as n from %s%s", quoteIdent(table), where)
		row, err := queryOne(qry, params...)
		if err != nil {
			writeError(w, err)
			return
		}
		n := int64(0)
		if row != nil {
			switch v := row["n"].(type) {
			case int64:
				n = v
			case float64:
				n = int64(v)
			case string:
				if i, e := strconv.ParseInt(v, 10, 64); e == nil {
					n = i
				}
			}
		}
		writeJSON(w, 200, map[string]any{"count": n})
		return
	}
	qry := fmt.Sprintf("select %s from %s%s%s", projection, quoteIdent(table), where, orderBy)
	values := append([]any{}, params...)
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n >= 0 {
			qry += fmt.Sprintf(" limit $%d", len(values)+1)
			values = append(values, n)
		} else {
			writeJSON(w, 400, map[string]any{"message": "limit tidak valid"})
			return
		}
	}
	if l := q.Get("offset"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n >= 0 {
			qry += fmt.Sprintf(" offset $%d", len(values)+1)
			values = append(values, n)
		} else {
			writeJSON(w, 400, map[string]any{"message": "offset tidak valid"})
			return
		}
	}
	rows, err := queryAll(qry, values...)
	if err != nil {
		writeError(w, err)
		return
	}
	casted, _ := castRows(table, rows)
	writeJSON(w, 200, stripUsersPassword(table, casted))
}

func decodeBody(r *http.Request) (any, error) {
	var body any
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func rowsOf(body any) []map[string]any {
	switch b := body.(type) {
	case []any:
		out := []map[string]any{}
		for _, item := range b {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{b}
	default:
		return nil
	}
}

func handleInsert(w http.ResponseWriter, r *http.Request, table string) {
	body, err := decodeBody(r)
	if err != nil {
		writeJSON(w, 400, map[string]any{"message": "JSON tidak valid"})
		return
	}
	rows := rowsOf(body)
	if len(rows) == 0 {
		writeJSON(w, 400, map[string]any{"message": "payload kosong"})
		return
	}
	var out []map[string]any
	for _, raw := range rows {
		prepared, err := prepareRow(table, raw)
		if err != nil {
			writeError(w, err)
			return
		}
		cols := keysOf(prepared)
		placeholders := make([]string, len(cols))
		values := make([]any, 0, len(cols))
		for i, c := range cols {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			values = append(values, prepared[c])
		}
		qry := fmt.Sprintf("insert into %s (%s) values (%s) returning *",
			quoteIdent(table), joinQuoted(cols), strings.Join(placeholders, ", "))
		inserted, err := queryAll(qry, values...)
		if err != nil {
			writeError(w, err)
			return
		}
		casted, _ := castRows(table, inserted)
		out = append(out, stripUsersPassword(table, casted)...)
	}
	writeJSON(w, 201, out)
}

func handleUpsert(w http.ResponseWriter, r *http.Request, table string) {
	q := r.URL.Query()
	conflictCol := q.Get("onConflict")
	if conflictCol == "" {
		conflictCol = "id"
	}
	colsDef, err := getTableColumns(table)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, ok := colsDef[conflictCol]; !ok {
		writeJSON(w, 400, pgErrJSON(missingColumnError(conflictCol, table)))
		return
	}
	body, err := decodeBody(r)
	if err != nil {
		writeJSON(w, 400, map[string]any{"message": "JSON tidak valid"})
		return
	}
	rows := rowsOf(body)
	if len(rows) == 0 {
		writeJSON(w, 400, map[string]any{"message": "payload kosong"})
		return
	}
	var out []map[string]any
	for _, raw := range rows {
		prepared, err := prepareRow(table, raw)
		if err != nil {
			writeError(w, err)
			return
		}
		cols := keysOf(prepared)
		values := make([]any, 0, len(cols))
		for i, c := range cols {
			_ = i
			values = append(values, prepared[c])
		}
		placeholders := make([]string, len(cols))
		for i := range cols {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		updateCols := []string{}
		for _, c := range cols {
			if c != conflictCol && c != "id" {
				updateCols = append(updateCols, fmt.Sprintf("%s = excluded.%s", quoteIdent(c), quoteIdent(c)))
			}
		}
		var qry string
		if len(updateCols) > 0 {
			qry = fmt.Sprintf("insert into %s (%s) values (%s) on conflict (%s) do update set %s returning *",
				quoteIdent(table), joinQuoted(cols), strings.Join(placeholders, ", "), quoteIdent(conflictCol), strings.Join(updateCols, ", "))
		} else {
			qry = fmt.Sprintf("insert into %s (%s) values (%s) on conflict (%s) do nothing returning *",
				quoteIdent(table), joinQuoted(cols), strings.Join(placeholders, ", "), quoteIdent(conflictCol))
		}
		rows2, err := queryAll(qry, values...)
		if err != nil {
			writeError(w, err)
			return
		}
		casted, _ := castRows(table, rows2)
		out = append(out, stripUsersPassword(table, casted)...)
	}
	writeJSON(w, 200, stripUsersPassword(table, out))
}

func handleUpdate(w http.ResponseWriter, r *http.Request, table string) {
	body, err := decodeBody(r)
	if err != nil {
		writeJSON(w, 400, map[string]any{"message": "JSON tidak valid"})
		return
	}
	raw, ok := body.(map[string]any)
	if !ok {
		writeJSON(w, 400, map[string]any{"message": "payload harus objek"})
		return
	}
	prepared, err := prepareRow(table, raw)
	if err != nil {
		writeError(w, err)
		return
	}
	if len(prepared) == 0 {
		writeJSON(w, 400, map[string]any{"message": "payload kosong"})
		return
	}
	cols := keysOf(prepared)
	startIndex := len(prepared)
	where, params, err := buildWhere(table, r.URL.Query(), startIndex)
	if err != nil {
		writeError(w, err)
		return
	}
	if where == "" {
		writeJSON(w, 400, map[string]any{"message": "update tanpa filter ditolak"})
		return
	}
	sets := make([]string, 0, len(cols))
	values := make([]any, 0, len(cols)+len(params))
	for i, c := range cols {
		sets = append(sets, fmt.Sprintf("%s = $%d", quoteIdent(c), i+1))
		values = append(values, prepared[c])
	}
	values = append(values, params...)
	qry := fmt.Sprintf("update %s set %s%s returning *", quoteIdent(table), strings.Join(sets, ", "), where)
	rows, err := queryAll(qry, values...)
	if err != nil {
		writeError(w, err)
		return
	}
	casted, _ := castRows(table, rows)
	writeJSON(w, 200, stripUsersPassword(table, casted))
}

func handleDelete(w http.ResponseWriter, r *http.Request, table string) {
	q := r.URL.Query()
	where, params, err := buildWhere(table, q, 0)
	if err != nil {
		writeError(w, err)
		return
	}
	if where == "" {
		writeJSON(w, 400, map[string]any{"message": "delete tanpa filter ditolak"})
		return
	}
	qry := fmt.Sprintf("delete from %s%s returning *", quoteIdent(table), where)
	rows, err := queryAll(qry, params...)
	if err != nil {
		writeError(w, err)
		return
	}
	casted, _ := castRows(table, rows)
	writeJSON(w, 200, stripUsersPassword(table, casted))
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// deterministic: sort
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func joinQuoted(cols []string) string {
	q := make([]string, len(cols))
	for i, c := range cols {
		q[i] = quoteIdent(c)
	}
	return strings.Join(q, ", ")
}

func pgErrJSON(e *pgError) map[string]any {
	return map[string]any{"message": e.Msg, "code": nullIfEmpty(e.Code), "details": nullIfEmpty(e.Detail), "hint": nullIfEmpty(e.Hint)}
}
