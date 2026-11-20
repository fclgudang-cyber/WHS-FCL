// accounts.go — kelola akun login FCL (tabel users) + import massal.
// Password selalu di-hash bcrypt; tidak pernah disimpan plaintext.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type fclAccountInput struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Nama       string `json:"nama"`
	Role       string `json:"role"`
	Divisi     string `json:"divisi"`
	Email      string `json:"email"`
	Permissions string `json:"permissions"`
	Status     string `json:"status"`
}

func h3dAccountJSON(w http.ResponseWriter, status int, v any) {
	writeJSON(w, status, v)
}

func accountClean(in fclAccountInput) fclAccountInput {
	in.Username = strings.TrimSpace(in.Username)
	in.Nama = strings.TrimSpace(in.Nama)
	in.Role = strings.TrimSpace(in.Role)
	in.Divisi = strings.TrimSpace(in.Divisi)
	in.Email = strings.TrimSpace(in.Email)
	in.Status = strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = "AKTIF"
	}
	if in.Permissions == "" {
		in.Permissions = "[]"
	}
	return in
}

func accountValidate(in fclAccountInput) string {
	if in.Username == "" {
		return "Username wajib diisi"
	}
	if len(in.Username) < 3 || len(in.Username) > 64 {
		return "Username harus 3-64 karakter"
	}
	if in.Password == "" {
		return "Password wajib diisi"
	}
	if len(in.Password) < 6 {
		return "Password minimal 6 karakter"
	}
	return ""
}

func accountExists(username string) (bool, error) {
	row, err := queryOne(`select id from users where lower(username) = lower($1) limit 1`, username)
	if err != nil {
		return false, err
	}
	return row != nil, nil
}

func accountInsert(in fclAccountInput) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("u-%d", time.Now().UnixNano())
	_, err = db.Exec(`insert into users
		(id, username, password, nama, role, divisi, email, permissions, status, "createdAt")
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, in.Username, string(hash), in.Nama, in.Role, in.Divisi, in.Email,
		in.Permissions, in.Status, time.Now().Format(time.RFC3339))
	return err
}

func handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var in fclAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h3dAccountJSON(w, 400, map[string]any{"success": false, "message": "JSON tidak valid"})
		return
	}
	in = accountClean(in)
	if msg := accountValidate(in); msg != "" {
		h3dAccountJSON(w, 400, map[string]any{"success": false, "message": msg})
		return
	}
	exists, err := accountExists(in.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	if exists {
		h3dAccountJSON(w, 409, map[string]any{"success": false, "message": "Username sudah digunakan"})
		return
	}
	if err = accountInsert(in); err != nil {
		writeError(w, err)
		return
	}
	h3dAccountJSON(w, 201, map[string]any{
		"success": true,
		"user": map[string]any{
			"username": in.Username, "nama": in.Nama, "role": in.Role,
			"divisi": in.Divisi, "email": in.Email, "status": in.Status,
		},
	})
}

func handleImportAccounts(w http.ResponseWriter, r *http.Request) {
	var rows []fclAccountInput
	if err := json.NewDecoder(r.Body).Decode(&rows); err != nil {
		h3dAccountJSON(w, 400, map[string]any{"success": false, "message": "JSON tidak valid"})
		return
	}
	type rowError struct {
		Row     int    `json:"row"`
		Message string `json:"message"`
	}
	created := 0
	skipped := 0
	errs := []rowError{}
	for i, raw := range rows {
		in := accountClean(raw)
		if msg := accountValidate(in); msg != "" {
			errs = append(errs, rowError{i + 2, msg})
			continue
		}
		exists, err := accountExists(in.Username)
		if err != nil {
			errs = append(errs, rowError{i + 2, err.Error()})
			continue
		}
		if exists {
			skipped++
			continue
		}
		if err = accountInsert(in); err != nil {
			errs = append(errs, rowError{i + 2, err.Error()})
			continue
		}
		created++
	}
	h3dAccountJSON(w, 200, map[string]any{
		"success": true, "created": created, "skipped": skipped, "errors": errs,
	})
}

func handleResetAccountPassword(w http.ResponseWriter, r *http.Request) {
	// path: /api/auth/account/{id}/reset-password
	part := strings.TrimPrefix(r.URL.Path, "/api/auth/account/")
	id := strings.TrimSuffix(part, "/reset-password")
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Password) < 6 {
		h3dAccountJSON(w, 400, map[string]any{"success": false, "message": "Password minimal 6 karakter"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := db.Exec(`update users set password = $1, "updatedAt" = $2 where id = $3`,
		string(hash), time.Now().Format(time.RFC3339), id)
	if err != nil {
		writeError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		h3dAccountJSON(w, 404, map[string]any{"success": false, "message": "Akun tidak ditemukan"})
		return
	}
	h3dAccountJSON(w, 200, map[string]any{"success": true, "message": "Password berhasil direset"})
}

func handleAccountTemplate(w http.ResponseWriter, _ *http.Request) {
	// Template CSV sederhana, bisa dibuka Excel / diisi ulang.
	header := "username,password,nama,role,divisi,email,status\n"
	example := "operator1,rahasia123,Operator Gudang,OPERATOR,WAREHOUSE,operator@fcl.local,AKTIF\n"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="template-akun-fcl.csv"`)
	_, _ = w.Write([]byte("\uFEFF" + header + example))
}
