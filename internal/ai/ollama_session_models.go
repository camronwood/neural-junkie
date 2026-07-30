package ai

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// sessionModelKey identifies a model at a specific Ollama endpoint.
type sessionModelKey struct {
	Endpoint string
	Model    string
}

var (
	sessionModelsMu sync.Mutex
	sessionModels   = map[sessionModelKey]struct{}{}
)

func normalizeOllamaEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "http://localhost:11434"
	}
	return endpoint
}

// NoteOllamaModelUsed records that Neural Junkie loaded or invoked an Ollama model.
// Tracked models are unloaded on hub/app exit.
func NoteOllamaModelUsed(endpoint, model string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return
	}
	key := sessionModelKey{
		Endpoint: normalizeOllamaEndpoint(endpoint),
		Model:    model,
	}
	sessionModelsMu.Lock()
	_, existed := sessionModels[key]
	if !existed {
		sessionModels[key] = struct{}{}
	}
	sessionModelsMu.Unlock()
	if !existed {
		persistSessionModels()
	}
}

// TrackedOllamaModels returns a copy of models noted during this process lifetime.
func TrackedOllamaModels() []sessionModelKey {
	sessionModelsMu.Lock()
	defer sessionModelsMu.Unlock()
	out := make([]sessionModelKey, 0, len(sessionModels))
	for k := range sessionModels {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Endpoint != out[j].Endpoint {
			return out[i].Endpoint < out[j].Endpoint
		}
		return out[i].Model < out[j].Model
	})
	return out
}

func clearTrackedOllamaModels() {
	sessionModelsMu.Lock()
	sessionModels = map[sessionModelKey]struct{}{}
	sessionModelsMu.Unlock()
	_ = os.Remove(sessionModelsPath())
}

func sessionModelsPath() string {
	if override := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SESSION_OLLAMA_MODELS")); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "session-ollama-models.json"
	}
	return filepath.Join(home, ".neural-junkie", "session-ollama-models.json")
}

type sessionModelsFile struct {
	Models []struct {
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	} `json:"models"`
}

func persistSessionModels() {
	tracked := TrackedOllamaModels()
	doc := sessionModelsFile{}
	for _, k := range tracked {
		doc.Models = append(doc.Models, struct {
			Endpoint string `json:"endpoint"`
			Model    string `json:"model"`
		}{Endpoint: k.Endpoint, Model: k.Model})
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	path := sessionModelsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, raw, 0o644)
}

// LoadPersistedSessionModels reads models recorded by a prior/current hub process
// (used by the desktop shell when unloading after a hard kill).
func LoadPersistedSessionModels() []sessionModelKey {
	raw, err := os.ReadFile(sessionModelsPath())
	if err != nil {
		return nil
	}
	var doc sessionModelsFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	out := make([]sessionModelKey, 0, len(doc.Models))
	seen := map[sessionModelKey]struct{}{}
	for _, row := range doc.Models {
		k := sessionModelKey{
			Endpoint: normalizeOllamaEndpoint(row.Endpoint),
			Model:    strings.TrimSpace(row.Model),
		}
		if k.Model == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

// UnloadTrackedOllamaModels soft-unloads every model Neural Junkie noted this session.
func UnloadTrackedOllamaModels(ctx context.Context) (unloaded []string, errs []error) {
	tracked := TrackedOllamaModels()
	if len(tracked) == 0 {
		// Fall back to disk so a parent process (Tauri) can still unload after kill.
		tracked = LoadPersistedSessionModels()
	}
	if len(tracked) == 0 {
		return nil, nil
	}
	for _, k := range tracked {
		unloadCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := UnloadOllamaModel(unloadCtx, k.Endpoint, k.Model)
		cancel()
		if err != nil {
			errs = append(errs, err)
			log.Printf("[ollama] unload session model %q @ %s: %v", k.Model, k.Endpoint, err)
			continue
		}
		unloaded = append(unloaded, k.Model)
	}
	clearTrackedOllamaModels()
	return unloaded, errs
}
