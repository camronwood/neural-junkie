package packs

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed all:builtin
var embeddedFS embed.FS

// BuiltinIDs are official pack ids shipped with the hub.
var BuiltinIDs = []string{"software-development", "life-sciences", "specialist-tuning"}

// LoadBuiltinManifest returns the embedded pack.yaml for an official pack id.
func LoadBuiltinManifest(packID string) (*Manifest, error) {
	path := filepath.Join("builtin", packID, "pack.yaml")
	data, err := embeddedFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("builtin pack %q: %w", packID, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// LoadBuiltinCatalog reads embedded builtin/catalog.json.
func LoadBuiltinCatalog() (*Catalog, error) {
	data, err := embeddedFS.ReadFile("builtin/catalog.json")
	if err != nil {
		return nil, fmt.Errorf("builtin catalog: %w", err)
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &cat, nil
}

// InstallBuiltin copies an embedded pack into destRoot/<packID>/.
func InstallBuiltin(packID, destRoot string) error {
	src := filepath.Join("builtin", packID)
	return fs.WalkDir(embeddedFS, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destRoot, packID, rel)
		if d.IsDir() {
			return osMkdirAll(dest)
		}
		data, err := embeddedFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := osMkdirAll(filepath.Dir(dest)); err != nil {
			return err
		}
		return osWriteFile(dest, data)
	})
}
