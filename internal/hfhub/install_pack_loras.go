package hfhub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func EnsureLoRAFiles(ctx context.Context, mgr *Manager, token, repoID, weightsFile string) error {
	for _, filename := range []string{weightsFile, AdapterConfigFilename} {
		if err := mgr.EnsureDownloadStarted(token, repoID, filename); err != nil {
			return err
		}
		if err := mgr.WatchDownload(ctx, repoID, filename, nil); err != nil && err != context.Canceled {
			return err
		}
	}
	return nil
}

// InstallLoRAResult is the outcome of one pack LoRA compose step.
type InstallLoRAResult struct {
	AgentType string `json:"agent_type,omitempty"`
	RepoID    string `json:"repo_id"`
	OllamaTag string `json:"ollama_tag"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// InstallPackLoRAs downloads (if needed) and composes pack-declared LoRA adapters.
func InstallPackLoRAs(ctx context.Context, mgr *Manager, manifest *packs.Manifest, token string) ([]InstallLoRAResult, error) {
	if mgr == nil {
		return nil, fmt.Errorf("HF manager not initialized")
	}
	if manifest == nil {
		return nil, fmt.Errorf("nil manifest")
	}
	var out []InstallLoRAResult
	for _, spec := range manifest.LoRAAdapters {
		res := InstallLoRAResult{
			AgentType: spec.AgentType,
			RepoID:    spec.RepoID,
			OllamaTag: spec.OllamaTag,
		}
		baseTag := strings.TrimSpace(spec.BaseOllamaTag)
		if baseTag == "" {
			baseTag = DefaultLoRABaseTag
		}
		tag := strings.TrimSpace(spec.OllamaTag)
		if tag == "" && spec.AgentType != "" {
			tag = SpecialistLoRATag(spec.AgentType)
		}
		res.OllamaTag = tag
		if !OllamaSafetensorLoRABaseSupported(baseTag) {
			res.Status = "skipped"
			res.Error = fmt.Sprintf(
				"base %q cannot be composed in Ollama (safetensors LoRA supports Llama, Mistral, Gemma only; use %s for code adapters)",
				baseTag,
				DefaultLoRATrainingCodeBase,
			)
			out = append(out, res)
			continue
		}
		filename := strings.TrimSpace(spec.Filename)
		if filename == "" {
			filename = "adapter_model.safetensors"
		}
		if err := EnsureLoRAFiles(ctx, mgr, token, spec.RepoID, filename); err != nil {
			res.Status = "error"
			res.Error = err.Error()
			out = append(out, res)
			continue
		}
		path, err := mgr.LocalPath(spec.RepoID, filename)
		if err != nil {
			res.Status = "error"
			res.Error = err.Error()
			out = append(out, res)
			continue
		}
		if err := ImportAdapterToOllama(ctx, baseTag, path, tag); err != nil {
			res.Status = "error"
			res.Error = err.Error()
			out = append(out, res)
			continue
		}
		res.OllamaTag = tag
		res.Status = "imported"
		out = append(out, res)
	}
	return out, nil
}
