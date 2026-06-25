package hub

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const generatedImagesDirName = "generated-images"

func saveGeneratedImageFile(messageID, mime, b64 string) (string, error) {
	messageID = strings.TrimSpace(messageID)
	b64 = strings.TrimSpace(b64)
	if messageID == "" || b64 == "" {
		return "", fmt.Errorf("message id and image data required")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decode generated image: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty generated image")
	}

	root, err := NeuralJunkieDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, generatedImagesDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	ext := generatedImageExt(mime)
	path := filepath.Join(dir, messageID+ext)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func generatedImageExt(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}

func decodeGeneratedImageBase64(b64 string) ([]byte, error) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return nil, fmt.Errorf("empty")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty decoded image")
	}
	return data, nil
}

func readGeneratedImageFile(path string) ([]byte, string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, "", fmt.Errorf("empty path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty file")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return data, "image/jpeg", nil
	case ".gif":
		return data, "image/gif", nil
	case ".webp":
		return data, "image/webp", nil
	default:
		return data, "image/png", nil
	}
}

func generatedImageFileForMessageID(messageID string) (string, bool) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return "", false
	}
	root, err := NeuralJunkieDataDir()
	if err != nil {
		return "", false
	}
	dir := filepath.Join(root, generatedImagesDirName)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		path := filepath.Join(dir, messageID+ext)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path, true
		}
	}
	return "", false
}
