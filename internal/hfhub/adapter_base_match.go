package hfhub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type adapterConfigJSON struct {
	BaseModelNameOrPath string `json:"base_model_name_or_path"`
}

// WarnAdapterBaseMismatch returns a non-fatal warning when adapter_config.json base
// does not obviously match the configured Ollama base tag.
func WarnAdapterBaseMismatch(baseTag, adapterDir string) string {
	baseTag = strings.TrimSpace(baseTag)
	adapterDir = strings.TrimSpace(adapterDir)
	if baseTag == "" || adapterDir == "" {
		return ""
	}
	configPath := filepath.Join(adapterDir, AdapterConfigFilename)
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var cfg adapterConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ""
	}
	hfBase := strings.ToLower(strings.TrimSpace(cfg.BaseModelNameOrPath))
	if hfBase == "" {
		return ""
	}
	ollama := normalizeOllamaTag(baseTag)
	// Heuristic: llama3.1:8b ↔ Meta-Llama-3.1-8B, llama3:8b ↔ Meta-Llama-3-8B, etc.
	if strings.Contains(hfBase, "llama-3.1") && strings.Contains(ollama, "llama3.1") {
		return ""
	}
	if strings.Contains(hfBase, "llama-3.2") && strings.Contains(ollama, "llama3.2") {
		return ""
	}
	if strings.Contains(hfBase, "llama-3") && (ollama == "llama3:8b" || strings.HasPrefix(ollama, "llama3:")) && !strings.Contains(ollama, "llama3.1") && !strings.Contains(ollama, "llama3.2") {
		return ""
	}
	if strings.Contains(hfBase, "mistral") && strings.Contains(ollama, "mistral") {
		return ""
	}
	return "adapter base_model_name_or_path " + cfg.BaseModelNameOrPath + " may not match Ollama base " + baseTag
}
