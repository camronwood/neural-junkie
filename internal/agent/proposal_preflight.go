package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProposalOperation is create or edit for preflight validation.
type ProposalOperation string

const (
	ProposalOpCreate ProposalOperation = "create"
	ProposalOpEdit   ProposalOperation = "edit"
)

// RedirectProposalPath maps common wrong paths to stack-manifest targets (e.g. misplaced tailwind config).
func RedirectProposalPath(path string, manifest *StackManifest) string {
	if manifest == nil {
		return normalizeFileChangeRelPath(path)
	}
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return path
	}
	lower := strings.ToLower(path)
	if strings.Contains(lower, "tailwind.config") && manifest.TailwindConfig != "" {
		if filepath.ToSlash(path) != manifest.TailwindConfig {
			return manifest.TailwindConfig
		}
	}
	base := strings.ToLower(filepath.Base(path))
	if (base == "app.vue" || base == "app.jsx") && manifest.EntryPoint != "" {
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
