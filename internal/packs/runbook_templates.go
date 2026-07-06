package packs

import (
	"os"
	"path/filepath"
	"strings"
)

// PackRunbookTemplateEntry is a JSON runbook template shipped in a pack.
type PackRunbookTemplateEntry struct {
	PackID string `json:"pack_id"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

// ListPackRunbookTemplates scans runbook_templates_glob for each enabled pack.
func ListPackRunbookTemplates(packDirs map[string]string, manifests map[string]*Manifest) ([]PackRunbookTemplateEntry, error) {
	var out []PackRunbookTemplateEntry
	for packID, dir := range packDirs {
		m := manifests[packID]
		if m == nil || strings.TrimSpace(m.Assets.RunbookTemplatesGlob) == "" {
			continue
		}
		paths, err := matchRunbooksGlob(dir, m.Assets.RunbookTemplatesGlob)
		if err != nil {
			continue
		}
		for _, rel := range paths {
			name := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
			out = append(out, PackRunbookTemplateEntry{PackID: packID, Path: rel, Name: name})
		}
	}
	return out, nil
}

// ReadPackRunbookTemplateJSON reads a pack runbook template file.
func ReadPackRunbookTemplateJSON(packDir, relPath string) ([]byte, error) {
	full := filepath.Join(packDir, relPath)
	return os.ReadFile(full)
}
