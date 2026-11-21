// Package fcl — backend FCL yang kini berjalan di dalam admin-service.
// Tidak ada lagi service/docker terpisah; rute /api/*, /iclock/* dan /fp
// dilayankan olehadmin-service pada port FCL_ADDR (default 6905).
package fcl

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if r.Method == http.MethodOptions {
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
	case p == "/api/file/upload" && r.Method == http.MethodPost:
		handleFileUpload(w, r)
	case strings.HasPrefix(p, "/api/files/pettycash/"):
		handleFileDownload(w, r)
	case p == "/iclock/cdata":
		handleADMS(w, r)
	case p == "/api/auth/login" && r.Method == http.MethodPost:
		handleLogin(w, r)
	case p == "/api/auth/change-password" && r.Method == http.MethodPost:
		handleChangePassword(w, r)
	case strings.HasPrefix(p, "/api/warehouse3d/"):
		handleWarehouse3D(w, r)
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

// Handler mengembalikan HTTP handler FCL (bisa dipasang ke server HTTP mana pun).
func Handler() http.Handler {
	return withCORS(router)
}

// Init menyiapkan koneksi DB dan menerapkan seluruh schema FCL.
func Init(dsn string) error {
	if dsn == "" {
		dsn = "host=postgres port=5432 user=postgres password=*Abcd123456 dbname=ghost_wms sslmode=disable"
	}
	if err := openDB(dsn); err != nil {
		return err
	}
	h3dApplySchema()
	return nil
}

// Start menjalankan HTTP server FCL di dalam proses admin-service.
func Start() (*http.Server, error) {
	if err := Init(os.Getenv("DATABASE_URL")); err != nil {
		return nil, err
	}
	addr := os.Getenv("FCL_ADDR")
	if addr == "" {
		addr = ":6905"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("fcl (admin-service) listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("fcl http server: %v", err)
		}
	}()
	return srv, nil
}

// Stop menghentikan server FCL secara graceful.
func Stop(srv *http.Server) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
