package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/packs"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
)

func waitForOllamaServer(ctx context.Context, timeout time.Duration) bool {
	if ollamaMgr == nil {
		return false
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ollamaMgr.IsServerRunning(ctx) {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(2 * time.Second):
		}
	}
	return ollamaMgr.IsServerRunning(ctx)
}

func packLoRAComposedTag(spec packs.LoRAAdapterSpec) string {
	tag := strings.TrimSpace(spec.OllamaTag)
	if tag == "" && spec.AgentType != "" {
		tag = hfhub.SpecialistLoRATag(spec.AgentType)
	}
	return tag
}

func packLoRAsInstalled(ctx context.Context, manifest *packs.Manifest) bool {
	if manifest == nil || len(manifest.LoRAAdapters) == 0 || ollamaMgr == nil {
		return true
	}
	for _, spec := range manifest.LoRAAdapters {
		tag := packLoRAComposedTag(spec)
		if tag == "" {
			continue
		}
		ok, err := ollamaMgr.HasModel(ctx, tag)
		if err != nil || !ok {
			return false
		}
	}
	return true
}

func ensurePackLoRABases(ctx context.Context, manifest *packs.Manifest) {
	if manifest == nil || ollamaMgr == nil {
		return
	}
	seen := make(map[string]struct{})
	for _, spec := range manifest.LoRAAdapters {
		base := strings.TrimSpace(spec.BaseOllamaTag)
		if base == "" {
			base = hfhub.DefaultLoRABaseTag
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		ok, err := ollamaMgr.HasModel(ctx, base)
		if err != nil {
			log.Printf("⚠️  pack LoRA: could not check base %s: %v", base, err)
			continue
		}
		if ok {
			continue
		}
		log.Printf("📥 pack LoRA: pulling base model %s", base)
		pullCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
		err = ollamaMgr.PullModel(pullCtx, base, func(p ollamaManager.PullProgress) {
			if p.Percent > 0 {
				log.Printf("   %s: %.1f%%", base, p.Percent)
			}
		})
		cancel()
		if err != nil {
			log.Printf("⚠️  pack LoRA pull %s failed: %v", base, err)
		}
	}
}

// ensurePackLoRAs downloads and composes pack-declared LoRA adapters for enabled packs.
// When onlyPackID is non-empty, only that pack is processed.
func ensurePackLoRAs(ctx context.Context, onlyPackID string) {
	if appConfig == nil || hfMgr == nil || ollamaMgr == nil {
		return
	}
	if !hasLoRACapability(capLoRAAdapters) {
		return
	}
	if !waitForOllamaServer(ctx, 2*time.Minute) {
		log.Printf("ℹ️  Ollama not running; skipping pack LoRA compose")
		return
	}

	manifests, err := appConfig.InstalledPackManifests()
	if err != nil {
		log.Printf("⚠️  pack LoRA: list installed packs: %v", err)
		return
	}

	token := hfhub.TokenFromConfig(appConfig)
	for _, manifest := range manifests {
		if manifest == nil {
			continue
		}
		if onlyPackID != "" && manifest.ID != onlyPackID {
			continue
		}
		if !appConfig.IsPackEnabled(manifest.ID) {
			continue
		}
		if len(manifest.LoRAAdapters) == 0 {
			continue
		}
		if packLoRAsInstalled(ctx, manifest) {
			continue
		}
		ensurePackLoRABases(ctx, manifest)
		log.Printf("📦 Ensuring pack LoRAs for %q ...", manifest.ID)
		installCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		results, err := hfhub.InstallPackLoRAs(installCtx, hfMgr, manifest, token)
		cancel()
		if err != nil {
			log.Printf("⚠️  pack LoRA install %q failed: %v", manifest.ID, err)
			continue
		}
		for _, res := range results {
			if res.Status == "imported" {
				log.Printf("   ✅ %s → %s", res.RepoID, res.OllamaTag)
			} else if res.Error != "" {
				log.Printf("   ⚠️  %s: %s", res.RepoID, res.Error)
			}
		}
	}
}

func triggerEnsurePackLoRAs(packID string) {
	go ensurePackLoRAs(context.Background(), strings.TrimSpace(packID))
}
