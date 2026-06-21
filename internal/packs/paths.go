package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UserPacksDir is ~/.neural-junkie/packs.
func UserPacksDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".neural-junkie", "packs"), nil
}

func osMkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// LoadInstalled scans UserPacksDir for pack.yaml manifests.
func LoadInstalled() ([]*Manifest, error) {
	root, err := UserPacksDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		m, err := LoadManifest(dir)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// InstalledPackDir returns ~/.neural-junkie/packs/<id>.
func InstalledPackDir(packID string) (string, error) {
	root, err := UserPacksDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, strings.TrimSpace(packID)), nil
}

// IsInstalled reports whether pack.yaml exists for packID.
func IsInstalled(packID string) bool {
	dir, err := InstalledPackDir(packID)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "pack.yaml"))
	return err == nil
}

// UninstallPack removes the installed pack directory.
func UninstallPack(packID string) error {
	dir, err := InstalledPackDir(packID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// ManifestByID finds a manifest in a slice.
func ManifestByID(manifests []*Manifest, id string) *Manifest {
	id = strings.TrimSpace(id)
	for _, m := range manifests {
		if m != nil && m.ID == id {
			return m
		}
	}
	return nil
}
