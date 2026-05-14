package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ImageGenerator is implemented by providers that can create images (e.g. OpenAI Images API).
type ImageGenerator interface {
	GenerateImage(ctx context.Context, prompt, size string) (mime string, base64Data string, err error)
}

type openAIImageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateImage calls OpenAI-compatible POST /v1/images/generations (response_format=b64_json).
func (p *OpenAICompatProvider) GenerateImage(ctx context.Context, prompt, size string) (string, string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", "", fmt.Errorf("empty prompt")
	}
	if size == "" {
		size = "1024x1024"
	}
	model := p.Model
	if model == "" || model == "default" {
		model = "dall-e-3"
	}
	body := map[string]interface{}{
		"model":            model,
		"prompt":           prompt,
		"n":                1,
		"size":             size,
		"response_format": "b64_json",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint+"/images/generations", bytes.NewBuffer(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}
	client := p.httpClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("images API status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed openAIImageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", fmt.Errorf("decode: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", "", fmt.Errorf("images API error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 || parsed.Data[0].B64JSON == "" {
		return "", "", fmt.Errorf("no image in response")
	}
	return "image/png", parsed.Data[0].B64JSON, nil
}
