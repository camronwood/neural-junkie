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
	RepoID      string        `json:"repo_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Tags        []string      `json:"tags"`
	SizeHint    string        `json:"size_hint,omitempty"`
	IconKey     string        `json:"icon_key,omitempty"`
	Publisher   string        `json:"publisher,omitempty"`
	Modes       []string      `json:"modes"` // "hosted", "local"
	Files       []CatalogFile `json:"files,omitempty"`
}

// Library returns the embedded catalog.
func Library() ([]LibraryModel, error) {
	var out []LibraryModel
	if err := json.Unmarshal(libraryJSON, &out); err != nil {
		return nil, fmt.Errorf("parse embedded hf library.json: %w", err)
	}
	return out, nil
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
