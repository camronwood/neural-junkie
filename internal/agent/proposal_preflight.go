package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/repo"
)

// ProposalOperation is create or edit for preflight validation.
type ProposalOperation string

const (
	ProposalOpCreate ProposalOperation = "create"
	ProposalOpEdit   ProposalOperation = "edit"
)

// stripScenarioBaselineSegments rewrites harness seed paths to the live workspace
// equivalent (e.g. .scenario-baseline/Makefile → Makefile).
func stripScenarioBaselineSegments(path string) string {
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return path
	}
	var out []string
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "" || part == repo.ScenarioBaselineDir {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "/")
}

// RedirectProposalPath maps common wrong paths to stack-manifest targets (e.g. misplaced tailwind config).
// Delete ops must not remapped App.js → App.tsx: boot-fix removes the corrupt App.js entry while
// keeping the real App.tsx entry (see attemptCorruptAppJSBootFix).
func RedirectProposalPath(path string, manifest *StackManifest) string {
	return RedirectProposalPathForOp(path, manifest, "")
}

// RedirectProposalPathForOp is RedirectProposalPath with an optional proposal operation
// ("create", "edit", "delete", …). Empty op keeps legacy create/edit redirect behavior.
func RedirectProposalPathForOp(path string, manifest *StackManifest, op string) string {
	return RedirectProposalPathForWorkspace(path, "", manifest, op)
}

// RedirectProposalPathForWorkspace redirects near-miss paths using the stack manifest and
// on-disk sibling presence (e.g. App.js decoy next to App.tsx entry).
func RedirectProposalPathForWorkspace(path, wsPath string, manifest *StackManifest, op string) string {
	path = stripScenarioBaselineSegments(path)
	if path == "" {
		return path
	}
	if manifest == nil {
		return path
	}
	lower := strings.ToLower(path)
	if strings.Contains(lower, "tailwind.config") && manifest.TailwindConfig != "" {
		if filepath.ToSlash(path) != manifest.TailwindConfig {
			return manifest.TailwindConfig
		}
	}
	if strings.EqualFold(strings.TrimSpace(op), "delete") {
		return path
	}
	base := strings.ToLower(filepath.Base(path))
	if (base == "app.vue" || base == "app.jsx" || base == "app.js") && manifest.EntryPoint != "" {
		entryBase := strings.ToLower(filepath.Base(manifest.EntryPoint))
		if base != entryBase && manifest.HasReact {
			entry := normalizeFileChangeRelPath(manifest.EntryPoint)
			// Prefer entry when it exists on disk, or when the decoy path exists alongside it.
			if wsPath == "" || workspaceFileExists(wsPath, entry) || workspaceFileExists(wsPath, path) {
				return entry
			}
			return entry
		}
	}
	return path
}

func workspaceFileExists(wsPath, rel string) bool {
	wsPath = strings.TrimSpace(wsPath)
	rel = normalizeFileChangeRelPath(rel)
	if wsPath == "" || rel == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(wsPath, filepath.FromSlash(rel)))
	return err == nil && !info.IsDir()
}

// DisambiguateAmbiguousBasename resolves bare/ambiguous basenames when multiple matches
// exist under the workspace. Prefers a path whose directory segment appears in userContent;
// otherwise returns an error listing candidates.
func DisambiguateAmbiguousBasename(wsPath, path, userContent string) (string, error) {
	path = normalizeFileChangeRelPath(path)
	if wsPath == "" || path == "" {
		return path, nil
	}
	// Multi-segment paths are explicit: keep existing files, and allow greenfield
	// creates (e.g. pkg/gamma/config.go) even when other config.go files exist.
	if strings.Contains(path, "/") {
		return path, nil
	}
	base := filepath.Base(path)
	if base == "" || base == "." || base == "/" {
		return path, nil
	}
	matches := findWorkspaceFilesNamed(wsPath, base)
	if len(matches) == 0 {
		return path, nil
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	lower := strings.ToLower(userContent)
	// Prefer an exact relative path mentioned in the user text.
	for _, m := range matches {
		if strings.Contains(lower, strings.ToLower(m)) {
			return m, nil
		}
	}
	// Else prefer the candidate whose most-specific directory segment appears
	// (ignore shared prefixes like "pkg" / "src").
	type scored struct {
		path  string
		score int
	}
	var best []scored
	for _, m := range matches {
		dir := strings.ToLower(filepath.ToSlash(filepath.Dir(m)))
		segs := strings.Split(dir, "/")
		score := 0
		for i, seg := range segs {
			seg = strings.TrimSpace(seg)
			if seg == "" || seg == "." || seg == "pkg" || seg == "src" || seg == "internal" {
				continue
			}
			if strings.Contains(lower, seg) {
				// Deeper segments score higher.
				score += i + 1
			}
		}
		if score > 0 {
			best = append(best, scored{path: m, score: score})
		}
	}
	if len(best) > 0 {
		top := best[0]
		uniqueTop := true
		for _, b := range best[1:] {
			if b.score > top.score {
				top = b
				uniqueTop = true
			} else if b.score == top.score && b.path != top.path {
				uniqueTop = false
			}
		}
		if uniqueTop {
			return top.path, nil
		}
	}
	return "", fmt.Errorf(
		"ambiguous path %q matches multiple files (%s) — use a full relative path or glob/list_dir to disambiguate",
		path, strings.Join(matches, ", "),
	)
}

func findWorkspaceFilesNamed(wsPath, baseName string) []string {
	wsPath = strings.TrimSpace(wsPath)
	baseName = strings.TrimSpace(baseName)
	if wsPath == "" || baseName == "" {
		return nil
	}
	want := strings.ToLower(baseName)
	var matches []string
	_ = filepath.WalkDir(wsPath, func(p string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == "target" || name == "dist" || name == ".scenario-baseline" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(name, want) {
			rel, relErr := filepath.Rel(wsPath, p)
			if relErr == nil {
				matches = append(matches, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	return matches
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ValidateProposal checks a file change path against workspace reality and stack manifest.
func ValidateProposal(wsPath, path string, op ProposalOperation, manifest *StackManifest) error {
	path = normalizeFileChangeRelPath(path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file path: %q", path)
	}
	wsPath = strings.TrimSpace(wsPath)
	if wsPath == "" {
		return nil
	}

	if manifest != nil {
		if err := validateProposalStackRules(path, manifest); err != nil {
			return err
		}
	}

	resolved := filepath.Join(wsPath, path)
	info, statErr := os.Stat(resolved)

	switch op {
	case ProposalOpEdit:
		if statErr != nil || info.IsDir() {
			if manifest != nil && manifest.TailwindConfig != "" && strings.Contains(strings.ToLower(path), "tailwind.config") {
				return fmt.Errorf("edit target does not exist: %q — use %q at repo root", path, manifest.TailwindConfig)
			}
			return fmt.Errorf("edit target does not exist: %q", path)
		}
		// Reject editing a decoy App.js when the real React entry is App.tsx (and exists).
		if manifest != nil {
			if err := validateEntryAlternateConflict(path, manifest); err != nil {
				if workspaceFileExists(wsPath, manifest.EntryPoint) {
					return err
				}
			}
		}
	case ProposalOpCreate:
		if statErr == nil && !info.IsDir() {
			// existing file — prefer edit; not a hard error
		}
		if manifest != nil {
			if err := validateEntryAlternateConflict(path, manifest); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateProposalStackRules(path string, manifest *StackManifest) error {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	if ext == ".vue" && !manifest.HasVue && manifest.ExtVue == 0 {
		return fmt.Errorf("path %q uses .vue but workspace has no Vue stack (react/tsx repo)", path)
	}

	if (base == "app.vue" || strings.HasSuffix(base, "app.vue")) && manifest.HasReact && manifest.EntryPoint != "" {
		entryExt := strings.ToLower(filepath.Ext(manifest.EntryPoint))
		if entryExt == ".tsx" || entryExt == ".jsx" {
			return fmt.Errorf("path %q is Vue entry but stack entry is %q", path, manifest.EntryPoint)
		}
	}

	if strings.Contains(strings.ToLower(path), "tailwind.config") && manifest.TailwindConfig != "" {
		norm := filepath.ToSlash(path)
		if norm != manifest.TailwindConfig {
			return fmt.Errorf("tailwind config belongs at %q, not %q", manifest.TailwindConfig, path)
		}
	}
	return nil
}

func validateEntryAlternateConflict(path string, manifest *StackManifest) error {
	if manifest == nil || manifest.EntryPoint == "" {
		return nil
	}
	base := strings.ToLower(filepath.Base(path))
	entryBase := strings.ToLower(filepath.Base(manifest.EntryPoint))
	if base == entryBase {
		return nil
	}
	alternates := map[string][]string{
		"app.tsx": {"app.js", "app.jsx"},
		"app.jsx": {"app.js", "app.tsx"},
		"app.js":  {"app.tsx", "app.jsx"},
	}
	for _, alt := range alternates[entryBase] {
		if base == alt {
			return fmt.Errorf(
				"path %q is not the project entry — edit %q instead (do not create a parallel App file)",
				path, manifest.EntryPoint,
			)
		}
	}
	return nil
}

// InferProposalOperation determines create vs edit from disk state.
func InferProposalOperation(wsPath, path string) ProposalOperation {
	path = normalizeFileChangeRelPath(path)
	if wsPath == "" || path == "" {
		return ProposalOpCreate
	}
	resolved := filepath.Join(wsPath, path)
	if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
		return ProposalOpEdit
	}
	return ProposalOpCreate
}
