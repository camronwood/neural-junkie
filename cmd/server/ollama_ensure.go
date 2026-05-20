package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
)

// ollamaTagsRequireHFImport cannot be installed with `ollama pull` (use Model Library → HF → Import to Ollama).
var ollamaTagsRequireHFImport = map[string]string{
	config.BioOllamaTag: "Model Library (⇧⌘M) → Hugging Face → Neural Junkie Bio 8B (GGUF) → Import to Ollama",
}

// ensureOllamaModels pulls configured tags when Ollama is running (background).
func ensureOllamaModels(ctx context.Context) {
	if appConfig == nil || ollamaMgr == nil {
		return
	}
	tags := appConfig.Ollama.ModelsToEnsure
	if len(tags) == 0 {
		return
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if ollamaMgr.IsServerRunning(ctx) {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !ollamaMgr.IsServerRunning(ctx) {
		log.Printf("ℹ️  Ollama not running; skipping models_to_ensure (%d tags)", len(tags))
		return
	}
	installed, err := ollamaMgr.ListModels(ctx)
	if err != nil {
		log.Printf("⚠️  Could not list Ollama models for models_to_ensure: %v", err)
		installed = nil
	}
	have := make(map[string]struct{}, len(installed))
	for _, name := range installed {
		have[name] = struct{}{}
	}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := have[tag]; ok {
			continue
		}
		if hint, skipPull := ollamaTagsRequireHFImport[tag]; skipPull {
			log.Printf("ℹ️  models_to_ensure: %s is not on the Ollama registry — install via %s", tag, hint)
			continue
		}
		log.Printf("📥 models_to_ensure: pulling %s", tag)
		pullCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
		err := ollamaMgr.PullModel(pullCtx, tag, func(p ollamaManager.PullProgress) {
			if p.Percent > 0 {
				log.Printf("   %s: %.1f%%", tag, p.Percent)
			}
		})
		cancel()
		if err != nil {
			log.Printf("⚠️  models_to_ensure pull %s failed: %v", tag, err)
		}
	}
}
