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
			return manifest.EntryPoint
		}
	}
	return path
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
