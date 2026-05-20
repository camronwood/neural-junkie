package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ollamaShowResponse struct {
	Capabilities []string `json:"capabilities"`
}

// OllamaBiologyFallbackModel is used when nj-bio returns empty replies (must be pulled in Ollama).
const OllamaBiologyFallbackModel = "qwen2.5:7b"

// OllamaModelPrefersCompactPrompt reports models that need a short system prompt (e.g. nj-bio GGUF).
func OllamaModelPrefersCompactPrompt(model string) bool {
	return ollamaModelLikelyNoNativeTools(model)
}

// PrefersCompactPrompt implements compact-prompt detection for agents.
func (o *OllamaProvider) PrefersCompactPrompt() bool {
	o.ensureNativeToolsKnown()
	return !o.nativeToolsSupported
}

// ollamaModelLikelyNoNativeTools is a fast path for known GGUF imports without tool templates.
func ollamaModelLikelyNoNativeTools(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.HasPrefix(m, "nj-bio") {
		return true
	}
	if strings.Contains(m, "openbiollm") {
		return true
	}
	if strings.HasPrefix(m, "koesn/") {
		return true
	}
	return false
}

func ollamaCapabilitiesIncludeTools(caps []string) bool {
	for _, c := range caps {
		if strings.EqualFold(strings.TrimSpace(c), "tools") {
			return true
		}
	}
	return false
}

func ollamaFetchCapabilities(ctx context.Context, endpoint, model string) ([]string, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	body, err := json.Marshal(map[string]string{"name": model})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/show", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama show status %d: %s", resp.StatusCode, string(raw))
	}
	var show ollamaShowResponse
	if err := json.Unmarshal(raw, &show); err != nil {
		return nil, err
	}
	return show.Capabilities, nil
}

func (o *OllamaProvider) ensureNativeToolsKnown() {
	o.toolsProbeOnce.Do(func() {
		o.nativeToolsSupported = true
		if ollamaModelLikelyNoNativeTools(o.Model) {
			o.nativeToolsSupported = false
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), ollamaHTTPTimeout(o.Model))
		defer cancel()
		caps, err := ollamaFetchCapabilities(ctx, o.Endpoint, o.Model)
		if err != nil {
			return
		}
		o.nativeToolsSupported = ollamaCapabilitiesIncludeTools(caps)
	})
}

// MarkNativeToolsUnsupported records that this model rejected tool calls (e.g. 400 from /api/chat).
func (o *OllamaProvider) MarkNativeToolsUnsupported() {
	o.ensureNativeToolsKnown()
	o.nativeToolsSupported = false
}
