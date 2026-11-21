// auth_middleware.go — Endpoint FCL memakai token JWT yang sama dengan
// sistem utama (diterbitkan core-service saat login).
package fcl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var fclJWTSecret = func() []byte {
	key := os.Getenv("JWT_SIGNING_KEY")
	if key == "" {
		// Harus sama dengan authenticator.yaml (placeholder default).
		key = "dev_only_change_me_in_prod"
	}
	return []byte(key)
}()

func publicFclPath(path string) bool {
	switch path {
	case "/api/health", "/fp", "/iclock/cdata",
		"/api/fingerprint-webhook", "/api/fingerprint-check":
		return true
	}
	return false
}

func requireJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if publicFclPath(r.URL.Path) || r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		header := r.Header.Get("Authorization")
		raw := ""
		if strings.HasPrefix(strings.ToLower(header), "bearer ") {
			raw = strings.TrimSpace(header[len("Bearer "):])
		} else {
			// EventSource tidak bisa mengirim header — izinkan token via query
			// khusus endpoint SSE.
			if strings.HasSuffix(r.URL.Path, "/activity/stream") {
				raw = strings.TrimSpace(r.URL.Query().Get("access_token"))
			}
		}
		if raw == "" {
			writeUnauthorized(w, "Token akses diperlukan")
			return
		}

		token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode tanda tangan tidak didukung")
			}
			return fclJWTSecret, nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
		if err != nil || token == nil || !token.Valid {
			writeUnauthorized(w, "Token akses tidak valid atau kedaluwarsa")
			return
		}
		next(w, r)
	}
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false, "message": message,
	})
}
