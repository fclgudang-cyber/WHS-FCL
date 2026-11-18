// adms.go — auto-registrasi mesin fingerprint yang terhubung via protokol ADMS.
// Mesin ZKTeco/ADMS biasanya memanggil GET/POST /iclock/cdata?SN=<serial>.
// SN + IP request disimpan ke fcl_api_devices agar muncul di "Daftar Mesin Fingerprint".
package main

import (
	"net"
	"net/http"
	"strings"
	"time"
)

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func registerADMSDevice(sn, ip string) {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return
	}
	var id string
	err := db.QueryRow(`select id from fcl_api_devices where lower(serialnumber) = lower($1) limit 1`, sn).Scan(&id)
	if err == nil && id != "" {
		// Perbarui IP bila berubah
		_, _ = db.Exec(`update fcl_api_devices set ipaddress = $1 where id = $2`, ip, id)
		return
	}
	_, _ = db.Exec(`insert into fcl_api_devices
		(id, nama, merk, ipaddress, serialnumber, aktif, keterangan, "createdAt")
		values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		"adms-"+strings.ToLower(sn), "ADMS "+sn, "Solution X900", ip, sn, "Ya",
		"Auto-terdaftar via ADMS", time.Now().Format(time.RFC3339))
}

func handleADMS(w http.ResponseWriter, r *http.Request) {
	sn := r.URL.Query().Get("SN")
	if sn == "" {
		sn = r.URL.Query().Get("sn")
	}
	if sn == "" && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
		_ = r.ParseForm()
		sn = r.FormValue("SN")
		if sn == "" {
			sn = r.FormValue("sn")
		}
	}
	registerADMSDevice(sn, clientIP(r))
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("OK\n"))
}
