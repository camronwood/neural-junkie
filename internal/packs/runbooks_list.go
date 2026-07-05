package packs

import (
	"os"
	"path/filepath"
	"strings"
)

// PackRunbookEntry is a markdown runbook shipped in a pack.
type PackRunbookEntry struct {
	PackID string `json:"pack_id"`
	Path   string `json:"path"`
	Title  string `json:"title"`
}

// ListPackRunbooks scans runbooks_glob for each enabled pack directory.
func ListPackRunbooks(packDirs map[string]string, manifests map[string]*Manifest) ([]PackRunbookEntry, error) {
	var out []PackRunbookEntry
	for packID, dir := range packDirs {
		m := manifests[packID]
		if m == nil || strings.TrimSpace(m.Assets.RunbooksGlob) == "" {
			continue
		}
		paths, err := matchRunbooksGlob(dir, m.Assets.RunbooksGlob)
		if err != nil {
			continue
		}
		for _, rel := range paths {
			title := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
			title = strings.ReplaceAll(title, "-", " ")
			out = append(out, PackRunbookEntry{PackID: packID, Path: rel, Title: title})
		}
	}
	return out, nil
}

// ReadPackRunbookMarkdown reads a pack runbook file relative to pack dir.
func ReadPackRunbookMarkdown(packDir, relPath string) (string, error) {
	full := filepath.Join(packDir, relPath)
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
