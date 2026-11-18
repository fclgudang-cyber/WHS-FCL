// signal.go — pengecekan konektivitas ke mesin fingerprint Solution X900.
// Browser tidak bisa melakukan raw TCP, jadi pengecekan dilakukan fcl-service.
package main

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type signalResult struct {
	IP        string `json:"ip"`
	Online    bool   `json:"online"`
	LatencyMs int64  `json:"latencyMs"`
	Port      int    `json:"port"`
}

func probeHost(ip string) signalResult {
	result := signalResult{IP: ip}
	ports := []int{80, 5000, 8080}
	for _, port := range ports {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, fmtInt(port)), 1200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			result.Online = true
			result.LatencyMs = time.Since(start).Milliseconds()
			result.Port = port
			return result
		}
	}
	return result
}

func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func handleFingerprintSignal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"success": false, "message": "method not allowed"})
		return
	}
	var body struct {
		IPs []string `json:"ips"`
		IP  string   `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "JSON tidak valid"})
		return
	}
	ips := body.IPs
	if len(ips) == 0 && body.IP != "" {
		ips = []string{body.IP}
	}
	if len(ips) > 20 {
		ips = ips[:20]
	}
	results := make([]signalResult, 0, len(ips))
	for _, ip := range ips {
		if ip == "" {
			continue
		}
		results = append(results, probeHost(ip))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "items": results})
}
