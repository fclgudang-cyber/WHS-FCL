// drive.go — upload bukti Petty Cash langsung ke Google Drive memakai OAuth 2.0 refresh token.
// Kredensial dibaca dari environment (file backend/deploy/fcl-drive.env, tidak masuk git).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultDriveFolderID = "1d1jqWSHxHQkEXu2bmhFDTTHuA36afRPd"

type driveConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	FolderID     string
}

func driveConfigFromEnv() driveConfig {
	folder := os.Getenv("GOOGLE_DRIVE_FOLDER_ID")
	if folder == "" {
		folder = defaultDriveFolderID
	}
	return driveConfig{
		ClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		RefreshToken: strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN")),
		FolderID:     folder,
	}
}

func (c driveConfig) enabled() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.RefreshToken != ""
}

func refreshGoogleAccessToken(cfg driveConfig) (string, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("refresh_token", cfg.RefreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out struct {
		AccessToken     string `json:"access_token"`
		Error           string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &out)
	if out.AccessToken == "" {
		detail := out.ErrorDescription
		if detail == "" {
			detail = out.Error
		}
		if detail == "" {
			detail = string(body)
		}
		return "", fmt.Errorf("gagal refresh Google access token: %s", detail)
	}
	return out.AccessToken, nil
}

func drivePermissionPublic(token, fileID string) {
	payload, _ := json.Marshal(map[string]string{"role": "reader", "type": "anyone"})
	req, err := http.NewRequest(
		http.MethodPost,
		"https://www.googleapis.com/drive/v3/files/"+url.PathEscape(fileID)+"/permissions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// ensureDriveFolder mencari (atau membuat) subfolder di dalam parent Drive.
func ensureDriveFolder(token, parentID, folderName string) (string, error) {
	folderName = strings.TrimSpace(folderName)
	if folderName == "" {
		return parentID, nil
	}
	client := &http.Client{Timeout: 20 * time.Second}
	escaped := strings.ReplaceAll(folderName, "'", "\\'")
	q := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and '%s' in parents and trashed=false", escaped, parentID)
	listURL := "https://www.googleapis.com/drive/v3/files?q=" + url.QueryEscape(q) + "&fields=files(id,name)&pageSize=10"
	req, err := http.NewRequest(http.MethodGet, listURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return "", err
	}
	var found struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	_ = json.Unmarshal(body, &found)
	if len(found.Files) > 0 && found.Files[0].ID != "" {
		return found.Files[0].ID, nil
	}

	meta, _ := json.Marshal(map[string]any{
		"name":     folderName,
		"mimeType": "application/vnd.google-apps.folder",
		"parents":  []string{parentID},
	})
	req2, err := http.NewRequest(http.MethodPost, "https://www.googleapis.com/drive/v3/files?fields=id,name", bytes.NewReader(meta))
	if err != nil {
		return "", err
	}
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req2)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", err
	}
	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return "", fmt.Errorf("gagal membuat folder Drive %q: %s", folderName, string(body2))
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body2, &created); err != nil || created.ID == "" {
		return "", fmt.Errorf("respons pembuatan folder Drive tidak valid: %s", string(body2))
	}
	return created.ID, nil
}

// uploadToDrive mengunggah file ke folder Drive (opsional subfolder periode) dan mengembalikan URL "view".
func uploadToDrive(cfg driveConfig, name string, data []byte, contentType, folderName string) (string, error) {
	token, err := refreshGoogleAccessToken(cfg)
	if err != nil {
		return "", err
	}
	parentID := cfg.FolderID
	if strings.TrimSpace(folderName) != "" {
		parentID, err = ensureDriveFolder(token, cfg.FolderID, folderName)
		if err != nil {
			return "", err
		}
	}
	meta, _ := json.Marshal(map[string]any{
		"name":    name,
		"parents": []string{parentID},
	})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	partMeta, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="metadata"`},
		"Content-Type":        []string{"application/json; charset=UTF-8"},
	})
	if err != nil {
		return "", err
	}
	_, _ = partMeta.Write(meta)
	ct := contentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	partFile, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{fmt.Sprintf(`form-data; name="media"; filename=%q`, name)},
		"Content-Type":        []string{ct},
	})
	if err != nil {
		return "", err
	}
	_, _ = partFile.Write(data)
	if err := w.Close(); err != nil {
		return "", err
	}

	endpoint := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id,webViewLink"
	req, err := http.NewRequest(http.MethodPost, endpoint, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gagal upload ke Google Drive (%d): %s", resp.StatusCode, string(body))
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil || created.ID == "" {
		return "", fmt.Errorf("respons Google Drive tidak valid: %s", string(body))
	}

	// Agar pratinjau <img> di Petty Cash bisa menampilkan foto langsung.
	drivePermissionPublic(token, created.ID)

	return fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=sharing", created.ID), nil
}
