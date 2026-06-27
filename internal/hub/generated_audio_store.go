package hub

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const generatedAudioDirName = "generated-audio"

func saveGeneratedAudioFile(messageID, mime, b64 string) (string, error) {
	messageID = strings.TrimSpace(messageID)
	b64 = strings.TrimSpace(b64)
	if messageID == "" || b64 == "" {
		return "", fmt.Errorf("message id and audio data required")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decode generated audio: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty generated audio")
	}

	root, err := NeuralJunkieDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, generatedAudioDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	ext := generatedAudioExt(mime)
	path := filepath.Join(dir, messageID+ext)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func generatedAudioExt(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/flac":
		return ".flac"
	case "audio/ogg":
		return ".ogg"
	default:
		return ".wav"
	}
}

func readGeneratedAudioFile(path string) ([]byte, string, error) {
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
	case ".mp3":
		return data, "audio/mpeg", nil
	case ".flac":
		return data, "audio/flac", nil
	case ".ogg":
		return data, "audio/ogg", nil
	default:
		return data, "audio/wav", nil
	}
}

func generatedAudioFileForMessageID(messageID string) (string, bool) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return "", false
	}
	root, err := NeuralJunkieDataDir()
	if err != nil {
		return "", false
	}
	dir := filepath.Join(root, generatedAudioDirName)
	for _, ext := range []string{".wav", ".mp3", ".flac", ".ogg"} {
		path := filepath.Join(dir, messageID+ext)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path, true
		}
	}
	return "", false
}
