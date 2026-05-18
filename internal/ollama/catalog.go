package ollama

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed library.json
var libraryJSON []byte

// LibraryModel is one curated row in the in-app model library (Ollama pull tag + metadata).
type LibraryModel struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	SizeHint    string   `json:"size_hint,omitempty"`
	IconKey     string   `json:"icon_key,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
}

// Library returns the embedded catalog (curated pull names and copy).
func Library() ([]LibraryModel, error) {
	var out []LibraryModel
	if err := json.Unmarshal(libraryJSON, &out); err != nil {
		return nil, fmt.Errorf("parse embedded library.json: %w", err)
	}
	return out, nil
}
