package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SyncPackFromDir validates srcDir and copies the pack into ~/.neural-junkie/packs/<id>/.
func SyncPackFromDir(srcDir string) (*Manifest, error) {
	srcDir = strings.TrimSpace(srcDir)
	if srcDir == "" {
		return nil, fmt.Errorf("pack directory required")
	}
	abs, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("pack directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
	}
	manifestDir, err := findManifestDir(abs)
	if err != nil {
		return nil, err
	}
	m, err := LoadManifest(manifestDir)
	if err != nil {
		return nil, err
	}
	destRoot, err := UserPacksDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return nil, err
	}
	dest := filepath.Join(destRoot, m.ID)
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := copyDir(manifestDir, dest); err != nil {
		return nil, err
	}
	return LoadManifest(dest)
}
