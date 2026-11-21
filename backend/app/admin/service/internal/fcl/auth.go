// auth.go — login / ubah password FCL (bcrypt $2a cocok dengan pgcrypto crypt()).
package fcl

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func verifyUser(username, password string) (map[string]any, error) {
	row, err := queryOne(`select id, username, password, nama, role, divisi, permissions, "createdAt" from users where lower(username) = lower($1) limit 1`, username)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	stored, _ := row["password"].(string)
	ok := false
	if stored == password {
		ok = true // plaintext lama
	} else if len(stored) > 4 && stored[:3] == "$2a" || len(stored) > 4 && stored[:3] == "$2b" || len(stored) > 4 && stored[:3] == "$2y" {
		if bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil {
			ok = true
		}
	}
	if !ok {
		return nil, nil
	}
	delete(row, "password")
	return row, nil
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	username, password := body["username"], body["password"]
	if username == "" || password == "" {
		writeJSON(w, 401, map[string]any{"success": false, "message": "Isi username dan password"})
		return
	}
	user, err := verifyUser(username, password)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if user == nil {
		writeJSON(w, 401, map[string]any{"success": false, "message": "Username atau password salah"})
		return
	}
	// permissions disimpan sebagai text JSON — kirim sebagai raw JSON string supaya frontend bisa JSON.parse
	writeJSON(w, 200, map[string]any{"success": true, "user": user})
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	username, oldPassword, newPassword := body["username"], body["oldPassword"], body["newPassword"]
	if username == "" || oldPassword == "" || newPassword == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Username, password lama, dan password baru wajib diisi"})
		return
	}
	user, err := verifyUser(username, oldPassword)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if user == nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": "Password lama salah"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	if _, err := db.Exec(`update users set password = $1 where username = $2`, string(hash), username); err != nil {
		writeJSON(w, 500, map[string]any{"success": false, "message": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"success": true, "message": "Password berhasil diubah"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if _, err := db.Exec("select 1"); err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "db": "down", "message": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "db": "up"})
}
