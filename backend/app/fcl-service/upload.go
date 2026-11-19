// upload.go — upload bukti transaksi (Petty Cash) langsung ke server.
// File disimpan di /app/uploads/pettycash dan dilayani lewat /api/files/.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const uploadDir = "/app/uploads/pettycash"

var safeFileName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func contentTypeToExt(ct string) string {
	c := strings.ToLower(strings.Split(ct, ";")[0])
	switch {
	case strings.Contains(c, "png"):
		return ".png"
	case strings.Contains(c, "jpeg"), strings.Contains(c, "jpg"):
		return ".jpg"
	case strings.Contains(c, "webp"):
		return ".webp"
	case strings.Contains(c, "gif"):
		return ".gif"
	case strings.Contains(c, "pdf"):
		return ".pdf"
	default:
		return ".bin"
	}
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12<<20) // maks 12 MB
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "file terlalu besar atau upload gagal"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "file tidak ditemukan"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "file kosong"})
		return
	}
	name := fmt.Sprintf("%d-%d%s", time.Now().UnixNano(), len(data)%100000, contentTypeToExt(header.Header.Get("Content-Type")))

	// Jika kredensial Google Drive tersedia, unggah langsung ke folder Drive.
	cfg := driveConfigFromEnv()
	if cfg.enabled() {
		driveName := header.Filename
		if driveName == "" || !safeFileName.MatchString(driveName) {
			driveName = name
		}
		periodName := strings.TrimSpace(r.FormValue("period"))
		driveURL, err := uploadToDrive(cfg, driveName, data, header.Header.Get("Content-Type"), periodName)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"success": false, "message": "Gagal upload ke Google Drive: " + err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true, "url": driveURL, "fileName": header.Filename,
			"size": len(data), "storage": "drive",
		})
		return
	}

	// Fallback: simpan lokal bila kredensial Drive belum dikonfigurasi.
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "message": "gagal membuat direktori upload"})
		return
	}
	if err := os.WriteFile(filepath.Join(uploadDir, name), data, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "message": "gagal menyimpan file"})
		return
	}
	url := "/api/files/pettycash/" + name
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true, "url": url, "fileName": header.Filename,
		"size": len(data), "storage": "local",
	})
}

func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/files/pettycash/")
	if name == "" || !safeFileName.MatchString(name) || strings.Contains(name, "..") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	path := filepath.Join(uploadDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ext := strings.ToLower(filepath.Ext(name))
	ct := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".webp": "image/webp", ".gif": "image/gif", ".pdf": "application/pdf",
	}[ext]
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}
