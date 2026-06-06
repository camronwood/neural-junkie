package hfhub

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed library.json
var libraryJSON []byte

// CatalogFile describes one downloadable or hosted HF model entry.
type CatalogFile struct {
	Filename string `json:"filename"`
	Quant    string `json:"quant,omitempty"`
	SizeHint string `json:"size_hint,omitempty"`
}

// LibraryModel is one row in the in-app HF model library.
type LibraryModel struct {
	Kind                   string        `json:"kind,omitempty"` // "full" (default) or "adapter"
	RepoID                 string        `json:"repo_id"`
	DownloadRepoID         string        `json:"download_repo_id,omitempty"` // Hub repo for GGUF when different from hosted repo_id
	Title                  string        `json:"title"`
	Description            string        `json:"description"`
	Tags                   []string      `json:"tags"`
	SizeHint               string        `json:"size_hint,omitempty"`
	IconKey                string        `json:"icon_key,omitempty"`
	Publisher              string        `json:"publisher,omitempty"`
	Modes                  []string      `json:"modes"` // "hosted", "local"
	Files                  []CatalogFile `json:"files,omitempty"`
	BaseOllamaTag          string        `json:"base_ollama_tag,omitempty"`
	DefaultOllamaTag       string        `json:"default_ollama_tag,omitempty"`
	AgentType              string        `json:"agent_type,omitempty"` // optional specialist slug for assign-to-agent UX
	Deprecated             bool          `json:"deprecated,omitempty"`
	OllamaComposeSupported *bool         `json:"ollama_compose_supported,omitempty"`
}

// Library returns the embedded catalog.
func Library() ([]LibraryModel, error) {
	var out []LibraryModel
	if err := json.Unmarshal(libraryJSON, &out); err != nil {
		return nil, fmt.Errorf("parse embedded hf library.json: %w", err)
	}
	for i := range out {
		if err := validateLibraryEntry(&out[i]); err != nil {
			return nil, fmt.Errorf("catalog entry %q: %w", out[i].RepoID, err)
		}
	}
	return out, nil
}

func validateLibraryEntry(entry *LibraryModel) error {
	if entry == nil {
		return fmt.Errorf("nil entry")
	}
	if strings.TrimSpace(entry.RepoID) == "" {
		return fmt.Errorf("repo_id is required")
	}
	if !IsAdapterEntry(entry) {
		return nil
	}
	if strings.TrimSpace(entry.BaseOllamaTag) == "" {
		return fmt.Errorf("adapter entries require base_ollama_tag")
	}
	if len(entry.Files) == 0 {
		return fmt.Errorf("adapter entries require at least one file")
	}
	hasLocal := false
	for _, m := range entry.Modes {
		if m == "local" {
			hasLocal = true
			break
		}
	}
	if !hasLocal {
		return fmt.Errorf("adapter entries require local mode")
	}
	return nil
}

// FindCatalogEntry returns a catalog row by repo_id (case-sensitive Hub id).
func FindCatalogEntry(repoID string) (*LibraryModel, error) {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return nil, fmt.Errorf("repo_id is required")
	}
	models, err := Library()
	if err != nil {
		return nil, err
	}
	for i := range models {
		if models[i].RepoID == repoID {
			return &models[i], nil
		}
	}
	return nil, fmt.Errorf("repo_id %q is not in the curated catalog", repoID)
}

// ResolveDownloadFilename picks filename from request or catalog default.
func ResolveDownloadFilename(entry *LibraryModel, filename string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename != "" {
		if entry != nil {
			if filename == AdapterConfigFilename && IsAdapterEntry(entry) {
				return filename, nil
			}
			for _, f := range entry.Files {
				if f.Filename == filename {
					return filename, nil
				}
			}
			return "", fmt.Errorf("filename %q is not allowed for %s", filename, entry.RepoID)
		}
		return filename, nil
	}
	if entry == nil || len(entry.Files) == 0 {
		return "", fmt.Errorf("filename is required")
	}
	return entry.Files[0].Filename, nil
}

// ResolveDownloadRepoID returns the Hub repo used for resolve/main GGUF downloads.
// Catalog repo_id is kept for hosted inference and on-disk cache keys.
func ResolveDownloadRepoID(entry *LibraryModel) string {
	if entry == nil {
		return ""
	}
	if id := strings.TrimSpace(entry.DownloadRepoID); id != "" {
		return id
	}
	return entry.RepoID
}

// AdapterOllamaComposeSupported reports whether a catalog adapter can be composed in Ollama.
func AdapterOllamaComposeSupported(entry *LibraryModel) bool {
	if entry == nil || !IsAdapterEntry(entry) {
		return false
	}
	if entry.Deprecated {
		return false
	}
	if entry.OllamaComposeSupported != nil {
		return *entry.OllamaComposeSupported
	}
	return OllamaSafetensorLoRABaseSupported(entry.BaseOllamaTag)
}
