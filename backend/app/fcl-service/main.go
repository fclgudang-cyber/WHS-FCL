// fcl-service — backend FCL (Go) untuk Ghost-main.
// Port dari server/ (Express) & laravel/ (Laravel) yang sudah terverifikasi.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func router(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case p == "/api/health":
		handleHealth(w, r)
	case p == "/api/fingerprint-webhook", p == "/fp":
		handleFingerprintWebhook(w, r)
	case p == "/api/fingerprint-check":
		handleFingerprintSignal(w, r)
	case p == "/iclock/cdata":
		handleADMS(w, r)
	case p == "/api/auth/login" && r.Method == "POST":
		handleLogin(w, r)
	case p == "/api/auth/change-password" && r.Method == "POST":
		handleChangePassword(w, r)
	case strings.HasPrefix(p, "/api/tables/"):
		table := tableName(r)
		switch r.Method {
		case http.MethodGet:
			handleList(w, r, table)
		case http.MethodPost:
			handleInsert(w, r, table)
		case http.MethodPut:
			handleUpsert(w, r, table)
		case http.MethodPatch:
			handleUpdate(w, r, table)
		case http.MethodDelete:
			handleDelete(w, r, table)
		default:
			writeJSON(w, 405, map[string]any{"message": "method not allowed"})
		}
	default:
		writeJSON(w, 404, map[string]any{"message": "endpoint tidak ditemukan"})
	}
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=ghost_wms sslmode=disable"
	}
	if err := openDB(dsn); err != nil {
		log.Fatal("fcl-service: koneksi database gagal:", err)
	}
	addr := os.Getenv("FCL_ADDR")
	if addr == "" {
		addr = ":6905"
	}
	log.Printf("fcl-service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(router)))
}
