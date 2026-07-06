package runbooklibrary

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// BundledDirCandidates returns directories that may contain bundled JSON templates.
func BundledDirCandidates(collabAssetsRoot string) []string {
	candidates := []string{
		filepath.Join("assets", "runbook-templates"),
		filepath.Join("neural-junkie", "assets", "runbook-templates"),
	}
	if root := strings.TrimSpace(collabAssetsRoot); root != "" {
		candidates = append([]string{filepath.Join(root, "runbook-templates")}, candidates...)
	}
	return candidates
}

func resolveBundledDir(collabAssetsRoot string) string {
	for _, d := range BundledDirCandidates(collabAssetsRoot) {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return BundledDirCandidates(collabAssetsRoot)[0]
}

// ListAllDefinitions aggregates user, bundled, and optional pack definitions.
func ListAllDefinitions(collabAssetsRoot string, packDefs []RunbookDefinition) ([]DefinitionSummary, error) {
	seen := map[string]bool{}
	var out []DefinitionSummary

	user, err := ListUserDefinitions()
	if err != nil {
		return nil, err
	}
	for _, s := range user {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		out = append(out, s)
	}

	for _, p := range packDefs {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		out = append(out, p.ToSummary())
	}

	dir := resolveBundledDir(collabAssetsRoot)
	templates, err := collaboration.ListRunbookTemplates(dir)
	if err != nil {
		return out, err
	}
	for _, t := range templates {
		def := FromTemplate(t, SourceBundled)
		if seen[def.ID] {
			continue
		}
		seen[def.ID] = true
		out = append(out, def.ToSummary())
	}
	if out == nil {
		return []DefinitionSummary{}, nil
	}
	return out, nil
}

// LoadDefinition resolves a definition by id from user library, pack defs, or bundled templates.
func LoadDefinition(id string, version int, collabAssetsRoot string, packDefs []RunbookDefinition) (*RunbookDefinition, error) {
	if version > 0 || userExists(id) {
		if def, err := LoadUserDefinition(id, version); err == nil {
			return def, nil
		} else if version > 0 {
			return nil, err
		}
	}
	for i := range packDefs {
		if packDefs[i].ID == id {
			d := packDefs[i]
			return &d, nil
		}
	}
	dir := resolveBundledDir(collabAssetsRoot)
	t, err := collaboration.LoadRunbookTemplate(dir, id)
	if err != nil {
		return nil, fmt.Errorf("definition %q not found", id)
	}
	def := FromTemplate(*t, SourceBundled)
	return &def, nil
}

func userExists(id string) bool {
	root, err := UserLibraryDir()
	if err != nil {
		return false
	}
	st, err := os.Stat(definitionDir(root, id))
	return err == nil && st.IsDir()
}
