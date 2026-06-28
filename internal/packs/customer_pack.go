package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CapabilityCustomerPack gates customer data packs (assets + settings overlay).
const CapabilityCustomerPack = "customer-pack"

// PackAssetsSpec locates bundled documentation and runbooks inside a pack directory.
type PackAssetsSpec struct {
	WorkspaceGuide string `yaml:"workspace_guide,omitempty"`
	RunbooksGlob   string `yaml:"runbooks_glob,omitempty"`
}

// IsCustomerPack reports whether the manifest is a customer data pack.
func (m *Manifest) IsCustomerPack() bool {
	if m == nil {
		return false
	}
	if m.HasCapability(CapabilityCustomerPack) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(m.PackKind), "customer")
}

// PackDir returns the installed pack directory for id.
func PackDir(packID string) (string, error) {
	return InstalledPackDir(packID)
}

// ResolvePackRelativePath resolves rel inside packDir (must stay within pack root).
func ResolvePackRelativePath(packDir, rel string) (string, error) {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute paths not allowed in pack assets: %q", rel)
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes pack directory: %q", rel)
	}
	abs := filepath.Join(packDir, clean)
	packAbs, err := filepath.Abs(packDir)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if targetAbs != packAbs && !strings.HasPrefix(targetAbs, packAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes pack directory: %q", rel)
	}
	return targetAbs, nil
}

// ReadWorkspaceGuide loads the workspace guide markdown/text for a customer pack.
func ReadWorkspaceGuide(m *Manifest, packDir string) (string, error) {
	if m == nil || strings.TrimSpace(m.Assets.WorkspaceGuide) == "" {
		return "", nil
	}
	path, err := ResolvePackRelativePath(packDir, m.Assets.WorkspaceGuide)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read workspace guide: %w", err)
	}
	return string(data), nil
}

// ResolveSettingsOverlay returns overlay values with pack-relative paths expanded to absolute.
func ResolveSettingsOverlay(m *Manifest, packDir string) (map[string]string, error) {
	if m == nil || len(m.SettingsOverlay) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(m.SettingsOverlay))
	for k, v := range m.SettingsOverlay {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" {
			continue
		}
		val = expandHomeInOverlay(val)
		if isPackRelativePathKey(key) && val != "" && !filepath.IsAbs(val) {
			abs, err := ResolvePackRelativePath(packDir, val)
			if err != nil {
				return nil, fmt.Errorf("settings_overlay[%q]: %w", key, err)
			}
			out[key] = abs
		} else {
			out[key] = val
		}
	}
	return out, nil
}

func expandHomeInOverlay(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	if strings.HasPrefix(val, "~") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			if val == "~" {
				return home
			}
			if strings.HasPrefix(val, "~/") || strings.HasPrefix(val, "~\\") {
				return filepath.Join(home, val[2:])
			}
		}
	}
	return os.ExpandEnv(val)
}

func isPackRelativePathKey(key string) bool {
	switch key {
	case "secondary_analysis_tools_path", "cumulative_qc_dir", "python_executable":
		return key != "python_executable"
	case "mcp.biology.secondary_analysis_tools_path", "mcp.biology.cumulative_qc_dir":
		return true
	default:
		return strings.HasSuffix(key, "_path") || strings.HasSuffix(key, "_dir")
	}
}
