package main

import (
	"context"
	"log"
	"math"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
)

// ollamaTagsRequireHFImport cannot be installed with `ollama pull` (use Model Library → HF → Import to Ollama).
var ollamaTagsRequireHFImport = map[string]string{
	config.BioOllamaTag: "Model Library (⇧⌘M) → Hugging Face → Neural Junkie Bio 8B (GGUF) → Import to Ollama",
}

func ollamaTagRequiresCompose(tag string) bool {
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, "nj-") {
		return false
	}
	// Composed LoRA tags: nj-security:14b, nj-biology:8b, nj-repo-*:14b, etc.
	if strings.Contains(tag, ":") {
		return true
	}
	return false
}

func listInstalledOllamaModels(ctx context.Context, mgr *ollamaManager.Manager) ([]string, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		installed, err := mgr.ListModels(ctx)
		if err == nil {
			return installed, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func logPullProgress(tag string, lastMilestone *float64) func(ollamaManager.PullProgress) {
	return func(p ollamaManager.PullProgress) {
		if p.Percent <= 0 {
			return
		}
		milestone := math.Floor(p.Percent/10.0) * 10
		if milestone > *lastMilestone || p.Percent >= 99 {
			log.Printf("   %s: %.0f%%", tag, p.Percent)
			*lastMilestone = milestone
		}
	}
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
	installed, err := listInstalledOllamaModels(ctx, ollamaMgr)
	if err != nil {
		log.Printf("⚠️  Could not list Ollama models for models_to_ensure; skipping pulls: %v", err)
		return
	}
	skipped := 0
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if ollamaManager.TagInstalled(installed, tag) {
			skipped++
			continue
		}
		if hint, skipPull := ollamaTagsRequireHFImport[tag]; skipPull {
			log.Printf("ℹ️  models_to_ensure: %s is not on the Ollama registry — install via %s", tag, hint)
			continue
		}
		if ollamaTagRequiresCompose(tag) {
			log.Printf("ℹ️  models_to_ensure: %s is a composed LoRA tag — ensured automatically from enabled pack LoRAs", tag)
			continue
		}
		log.Printf("📥 models_to_ensure: pulling %s", tag)
		var lastMilestone float64 = -1
		pullCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
		err := ollamaMgr.PullModel(pullCtx, tag, logPullProgress(tag, &lastMilestone))
		cancel()
		if err != nil {
			log.Printf("⚠️  models_to_ensure pull %s failed: %v", tag, err)
			continue
		}
		installed = append(installed, tag)
	}
	if skipped > 0 {
		log.Printf("ℹ️  models_to_ensure: %d tag(s) already installed", skipped)
	}
}
