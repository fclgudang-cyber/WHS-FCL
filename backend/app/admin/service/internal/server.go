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

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	if _, err := db.Exec("select 1"); err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "db": "down", "message": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "db": "up"})
}

// Handler mengembalikan HTTP handler FCL dengan JWT utama + CORS.
func Handler() http.Handler {
	return withCORS(requireJWT(router))
}

// webhookRouter hanya membuka endpoint mesin fingerprint (tanpa JWT).
func webhookRouter(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case p == "/api/health":
		handleHealth(w, r)
	case p == "/fp", p == "/api/fingerprint-webhook":
		handleFingerprintWebhook(w, r)
	case p == "/api/fingerprint-check":
		handleFingerprintSignal(w, r)
	case p == "/iclock/cdata":
		handleADMS(w, r)
	default:
		writeJSON(w, 404, map[string]any{"message": "endpoint webhook tidak ditemukan"})
	}
}

func WebhookHandler() http.Handler {
	return withCORS(webhookRouter)
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

// Start menjalankan dua listener:
//   - FCL_ADDR (default :6905): seluruh API FCL dengan JWT utama.
//   - FCL_WEBHOOK_ADDR (default :80): khusus webhook/ADMS fingerprint tanpa JWT.
func Start() ([]*http.Server, error) {
	if err := Init(os.Getenv("DATABASE_URL")); err != nil {
		return nil, err
	}
	type listener struct {
		name    string
		addr    string
		handler http.Handler
	}
	fclAddr := os.Getenv("FCL_ADDR")
	if fclAddr == "" {
		fclAddr = ":6905"
	}
	webhookAddr := os.Getenv("FCL_WEBHOOK_ADDR")
	if webhookAddr == "" {
		webhookAddr = ":80"
	}
	listeners := []listener{
		{"fcl", fclAddr, Handler()},
		{"fingerprint-webhook", webhookAddr, WebhookHandler()},
	}
	servers := make([]*http.Server, 0, len(listeners))
	for _, l := range listeners {
		srv := &http.Server{
			Addr:              l.addr,
			Handler:           l.handler,
			ReadHeaderTimeout: 10 * time.Second,
		}
		servers = append(servers, srv)
		go func(name, addr string, srv *http.Server) {
			log.Printf("fcl %s listening on %s", name, addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("fcl %s (%s) berhenti: %v", name, addr, err)
			}
		}(l.name, l.addr, srv)
	}
	return servers, nil
}

// Stop menghentikan semua listener FCL secara graceful.
func Stop(servers []*http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range servers {
		if srv != nil {
			_ = srv.Shutdown(ctx)
		}
	}
}
