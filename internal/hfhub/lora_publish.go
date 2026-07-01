package hfhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PublishAdapter uploads adapter weights to a Hugging Face repo.
func PublishAdapter(ctx context.Context, token, repoID, adapterDir string) error {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return fmt.Errorf("repo_id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("hf token required for publish")
	}
	adapterDir = strings.TrimSpace(adapterDir)
	weights := filepath.Join(adapterDir, "adapter_model.safetensors")
	if _, err := os.Stat(weights); err != nil {
		return fmt.Errorf("adapter weights missing: %w", err)
	}
	config := filepath.Join(adapterDir, "adapter_config.json")
	files := []string{weights}
	if st, err := os.Stat(config); err == nil && !st.IsDir() {
		files = append(files, config)
	}
	for _, f := range files {
		if err := uploadHFRepoFile(ctx, token, repoID, filepath.Base(f), f); err != nil {
			return err
		}
	}
	return nil
}

func uploadHFRepoFile(ctx context.Context, token, repoID, destName, localPath string) error {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", destName)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	if _, err := part.Write(raw); err != nil {
		return err
	}
	_ = w.WriteField("path", destName)
	if err := w.Close(); err != nil {
		return err
	}
	url := fmt.Sprintf("https://huggingface.co/api/repos/%s/commit/main", repoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	payload, _ := json.Marshal(map[string]any{
		"operations": []map[string]string{{"operation": "addOrUpdate", "path": destName}},
	})
	req.Header.Set("X-Commit-Operations", string(payload))
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hf upload %s: %s %s", destName, resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}
