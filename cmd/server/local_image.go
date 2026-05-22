package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleLocalImage serves a local image file for the desktop UI when history redacts base64 blobs.
// Local hub only; paths must be under the user's home directory.
func handleLocalImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("path"))
	clean, err := validateLocalImagePath(raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch strings.ToLower(filepath.Ext(clean)) {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "image/png")
	}
	http.ServeFile(w, r, clean)
}

func validateLocalImagePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errLocalImagePath("path required")
	}
	clean := filepath.Clean(p)
	if !filepath.IsAbs(clean) {
		return "", errLocalImagePath("path must be absolute")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	home = filepath.Clean(home)
	rel, err := filepath.Rel(home, clean)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errLocalImagePath("path must be under home directory")
	}
	switch strings.ToLower(filepath.Ext(clean)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
	default:
		return "", errLocalImagePath("unsupported image type")
	}
	st, err := os.Stat(clean)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", errLocalImagePath("not a file")
	}
	return clean, nil
}

type localImagePathError string

func (e localImagePathError) Error() string { return string(e) }

func errLocalImagePath(msg string) error { return localImagePathError(msg) }
