package runbooklibrary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var storeMu sync.Mutex

// SaveUserDefinition persists a new version of a user definition.
func SaveUserDefinition(def RunbookDefinition) (*RunbookDefinition, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	root, err := UserLibraryDir()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(def.ID) == "" {
		def.ID = uuid.New().String()
	}
	def.Source = SourceUser
	dir := definitionDir(root, def.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	manifest := Manifest{ID: def.ID, Title: def.Title}
	if b, err := os.ReadFile(manifestPath(root, def.ID)); err == nil {
		_ = json.Unmarshal(b, &manifest)
	}
	if def.Version <= 0 {
		def.Version = manifest.LatestVersion + 1
	}
	if def.Version <= manifest.LatestVersion {
		def.Version = manifest.LatestVersion + 1
	}
	if def.Title == "" {
		def.Title = manifest.Title
	}
	if def.Title == "" {
		def.Title = def.ID
	}
	def.UpdatedAt = time.Now().UTC()

	body, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := atomicWrite(versionPath(root, def.ID, def.Version), body); err != nil {
		return nil, err
	}
	manifest.Title = def.Title
	manifest.LatestVersion = def.Version
	manifest.UpdatedAt = def.UpdatedAt
	mb, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := atomicWrite(manifestPath(root, def.ID), mb); err != nil {
		return nil, err
	}
	out := def
	return &out, nil
}

// LoadUserDefinition loads a user definition by id and optional version (0 = latest).
func LoadUserDefinition(id string, version int) (*RunbookDefinition, error) {
	root, err := UserLibraryDir()
	if err != nil {
		return nil, err
	}
	dir := definitionDir(root, id)
	if version <= 0 {
		b, err := os.ReadFile(manifestPath(root, id))
		if err != nil {
			return nil, fmt.Errorf("definition %q not found: %w", id, err)
		}
		var m Manifest
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		version = m.LatestVersion
	}
	path := versionPath(root, id, version)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("definition %q v%d not found: %w", id, version, err)
	}
	var def RunbookDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	if def.Source == "" {
		def.Source = SourceUser
	}
	_ = dir
	return &def, nil
}

// DeleteUserDefinition removes a user definition directory.
func DeleteUserDefinition(id string) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	root, err := UserLibraryDir()
	if err != nil {
		return err
	}
	dir := definitionDir(root, id)
	return os.RemoveAll(dir)
}

// ListUserDefinitions returns summaries for all user definitions.
func ListUserDefinitions() ([]DefinitionSummary, error) {
	root, err := UserLibraryDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []DefinitionSummary{}, nil
		}
		return nil, err
	}
	var out []DefinitionSummary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mb, err := os.ReadFile(filepath.Join(root, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(mb, &m); err != nil {
			continue
		}
		def, err := LoadUserDefinition(m.ID, m.LatestVersion)
		if err != nil {
			out = append(out, DefinitionSummary{
				ID: m.ID, Title: m.Title, Version: m.LatestVersion,
				Source: SourceUser, UpdatedAt: m.UpdatedAt,
			})
			continue
		}
		out = append(out, def.ToSummary())
	}
	if out == nil {
		return []DefinitionSummary{}, nil
	}
	return out, nil
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".runbook-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
