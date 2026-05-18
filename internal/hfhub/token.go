package hfhub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
)

// TokenFromConfig returns HF API token from provider rows, HF config, or environment.
func TokenFromConfig(cfg *config.Config) string {
	if cfg == nil {
		return ai.ResolveHFToken("")
	}
	for _, p := range cfg.ListProvidersSnapshot() {
		if p.Type == "huggingface" && strings.TrimSpace(p.APIKey) != "" {
			return strings.TrimSpace(p.APIKey)
		}
	}
	if t := strings.TrimSpace(cfg.HF.Token); t != "" {
		return t
	}
	return ai.ResolveHFToken("")
}
