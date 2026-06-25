package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// UnloadOllamaModel removes a model from Ollama memory (POST /api/generate, keep_alive: 0).
func UnloadOllamaModel(ctx context.Context, endpoint, model string) error {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model name required")
	}
	body, err := json.Marshal(map[string]interface{}{
		"model":      model,
		"keep_alive": 0,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama unload status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ollamaImageUnloadAfterUse reports whether to unload the image model after each generation.
// Default true. Set OLLAMA_IMAGE_KEEP_ALIVE=-1 (or NEURAL_JUNKIE_IMAGE_KEEP_ALIVE) to keep loaded.
func ollamaImageUnloadAfterUse() bool {
	raw := strings.TrimSpace(os.Getenv("OLLAMA_IMAGE_KEEP_ALIVE"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_IMAGE_KEEP_ALIVE"))
	}
	if raw == "" || raw == "0" {
		return true
	}
	if raw == "-1" || strings.HasPrefix(raw, "-") {
		return false
	}
	// Positive duration (e.g. 5m): let Ollama retain the model; skip explicit unload.
	return false
}
