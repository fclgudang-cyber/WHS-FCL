// webhook.go — penerima push absensi dari mesin fingerprint Solution X900 (ADMS)
// dan sinkronisasi ke tabel absensikaryawan + master karyawan.
package fcl

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func str(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case json.Number:
		return x.String()
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", x))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func first(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if s := str(v); s != "" {
				return s
			}
		}
	}
	return ""
}

// normalizeFingerprintRecords menerima tiga bentuk payload:
//   - array langsung: [ {...}, {...} ]
//   - { "records": [...] } / { "data": [...] }
//   - format Google Apps Script: { "func": "syncFingerprintData", "args": [[ {...} ]] }
func normalizeFingerprintRecords(body any) []map[string]any {
	var raw []any
	switch b := body.(type) {
	case []any:
		raw = b
	case map[string]any:
		if f := str(b["func"]); f != "" && strings.Contains(strings.ToLower(f), "fingerprint") {
			if args, ok := b["args"].([]any); ok && len(args) > 0 {
				if inner, ok := args[0].([]any); ok {
					raw = inner
				} else if inner, ok := args[0].(map[string]any); ok {
					raw = []any{inner}
				}
			}
		}
		if raw == nil {
			if r, ok := b["records"].([]any); ok {
				raw = r
			} else if r, ok := b["data"].([]any); ok {
				raw = r
			}
		}
	default:
		return nil
	}

	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case map[string]any:
			out = append(out, v)
		case []any:
			for _, sub := range v {
				if m, ok := sub.(map[string]any); ok {
					out = append(out, m)
				}
			}
		}
	}
	return out
}

func handleFingerprintWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}
	var body any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "JSON tidak valid"})
		return
	}

	records := normalizeFingerprintRecords(body)
	if len(records) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "Tidak ada data fingerprint yang diterima"})
		return
	}

	// Master karyawan: index by id, fingerprintId, dan nama
	type master struct {
		id, nama, divisi, jabatan, fp string
	}
	masterByKey := map[string]master{}
	if rows, err := queryAll(`select id, nama, divisi, jabatan, "fingerprintId" from karyawan`); err == nil {
		for _, row := range rows {
			m := master{
				id: str(row["id"]), nama: str(row["nama"]),
				divisi: str(row["divisi"]), jabatan: str(row["jabatan"]),
				fp: str(row["fingerprintId"]),
			}
			if m.id != "" {
				masterByKey["id:"+strings.ToLower(m.id)] = m
			}
			if m.fp != "" {
				masterByKey["fp:"+strings.ToLower(m.fp)] = m
			}
			if m.nama != "" {
				masterByKey["nama:"+strings.ToLower(m.nama)] = m
			}
		}
	}

	// Dedupe terhadap data absen yang sudah ada
	seen := map[string]bool{}
	if rows, err := queryAll(`select tanggal, "fingerprintId", jam, tipe from absensikaryawan`); err == nil {
		for _, row := range rows {
			seen[str(row["tanggal"])+"|"+strings.ToLower(str(row["fingerprintId"]))+"|"+str(row["jam"])+"|"+strings.ToUpper(str(row["tipe"]))] = true
		}
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil || loc == nil {
		loc = time.FixedZone("WIB", 7*60*60)
	}
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")

	added, skipped := 0, 0
	for _, rec := range records {
		rawFp := first(rec, "fingerprintId", "finger_id", "fpId")
		karID := first(rec, "karyawanId", "user_id", "karyawanid")
		tanggal := first(rec, "tanggal", "date")
		if len(tanggal) > 10 {
			tanggal = tanggal[:10]
		}
		if tanggal == "" {
			tanggal = today
		}
		jam := first(rec, "jam", "time")
		tipe := strings.ToUpper(first(rec, "tipe", "inout", "direction", "status"))
		if tipe == "" {
			tipe = "IN"
		}
		if tipe == "0" {
			tipe = "IN"
		}
		if tipe == "1" {
			tipe = "OUT"
		}
		if tipe == "CHECKIN" || tipe == "MASUK" {
			tipe = "IN"
		}
		if tipe == "CHECKOUT" || tipe == "PULANG" || tipe == "KELUAR" {
			tipe = "OUT"
		}

		key := tanggal + "|" + strings.ToLower(rawFp) + "|" + jam + "|" + tipe
		if seen[key] {
			skipped++
			continue
		}
		seen[key] = true

		// Cari master by id → fingerprint → nama → karyawanId dari payload
		var m master
		if found, ok := masterByKey["id:"+strings.ToLower(karID)]; ok {
			m = found
		} else if found, ok := masterByKey["fp:"+strings.ToLower(rawFp)]; ok {
			m = found
		} else if found, ok := masterByKey["nama:"+strings.ToLower(first(rec, "nama", "name"))]; ok {
			m = found
		}
		if m.id != "" {
			karID = m.id
		}
		nama := m.nama
		if nama == "" {
			nama = first(rec, "nama", "name")
		}
		if nama == "" {
			nama = "FP-" + rawFp
		}
		divisi := m.divisi
		if divisi == "" {
			divisi = first(rec, "divisi", "department")
		}
		if divisi == "" {
			divisi = "Tidak Diketahui"
		}
		jabatan := m.jabatan
		if jabatan == "" {
			jabatan = first(rec, "jabatan", "position")
		}
		fpID := rawFp
		if fpID == "" {
			fpID = m.fp
		}
		status := "Hadir"
		if tipe == "IN" && jam > "08:15:00" {
			status = "Terlambat"
		}

		buf := make([]byte, 4)
		_, _ = rand.Read(buf)
		id := fmt.Sprintf("%d-%s", now.Unix(), hex.EncodeToString(buf))
		_, err := db.Exec(`insert into absensikaryawan
			(id, tanggal, jam, "karyawanId", nama, divisi, jabatan, tipe, sumber, "fingerprintId", status, keterangan, "createdAt")
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			id, tanggal, jam, karID, nama, divisi, jabatan, tipe, "fingerprint", fpID, status,
			first(rec, "keterangan", "remark"), now.Format(time.RFC3339))
		if err != nil {
			logWebhookError(err)
			continue
		}
		added++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true, "added": added, "skipped": skipped, "total": len(records),
	})
}

func logWebhookError(err error) {
	fmt.Println("fingerprint-webhook insert error:", err)
}
