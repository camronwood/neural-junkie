package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// ValidatePackRequestContext builds a PackRequirementsContext from hub config.
func (c *Config) ValidatePackRequestContext() *packs.PackRequirementsContext {
	if c == nil {
		return nil
	}
	return &packs.PackRequirementsContext{
		IsInstalled: func(packID string) bool {
			return c.IsPackInstalled(packID)
		},
		IsEnabled: func(packID string) bool {
			return c.IsPackEnabled(packID)
		},
		EnabledCapabilities: func() []string {
			return c.EnabledCapabilities()
		},
	}
}

// ValidatePackZip dry-runs a zip without installing.
func (c *Config) ValidatePackZip(data []byte) (*packs.ValidationReport, error) {
	return packs.ValidateZipBytes(data, c.ValidatePackRequestContext())
}

// ValidatePackDir dry-runs a local pack folder without installing.
func (c *Config) ValidatePackDir(dir string) (*packs.ValidationReport, error) {
	return packs.ValidatePackDir(dir, c.ValidatePackRequestContext())
}

// ValidatePackYAML dry-runs pack.yaml text with optional asset root.
func (c *Config) ValidatePackYAML(yamlText, assetRoot string) (*packs.ValidationReport, error) {
	return packs.ValidateYAML(yamlText, assetRoot, c.ValidatePackRequestContext())
}

// DevLinkPack syncs a local folder into installed packs and records dev_sources (does not enable).
// Official catalog pack ids are allowed when linking the real pack repo (not a customer pack).
func (c *Config) DevLinkPack(srcDir string) (*packs.Manifest, error) {
	srcDir = strings.TrimSpace(srcDir)
	if srcDir == "" {
		return nil, fmt.Errorf("pack_dir required")
	}
	abs, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, err
	}
	pre, err := packs.LoadManifest(abs)
	if err != nil {
		// Manifest may live one level down; SyncPackFromDir finds it — validate after sync path.
		pre = nil
	}
	if pre == nil || !packs.IsOfficialPackID(pre.ID) {
		report, err := c.ValidatePackDir(abs)
		if err != nil {
			return nil, err
		}
		if report != nil && !report.Valid {
			msg := "pack validation failed"
			if len(report.Errors) > 0 {
				msg = report.Errors[0]
			}
			return nil, fmt.Errorf("%s", msg)
		}
	} else if pre.IsCustomerPack() {
		return nil, fmt.Errorf("pack id %q collides with an official catalog pack", pre.ID)
	}
	m, err := packs.SyncPackFromDir(abs)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	if c.Packs.DevSources == nil {
		c.Packs.DevSources = make(map[string]string)
	}
	wasEnabled := c.packEnabledLocked(m.ID)
	if !c.packInstalledLocked(m.ID) {
		c.Packs.Installed = append(c.Packs.Installed, m.ID)
		c.Packs.Enabled[m.ID] = false
	} else if !wasEnabled {
		c.Packs.Enabled[m.ID] = false
	}
	c.Packs.DevSources[m.ID] = abs
	return m, nil
}

// DevReloadPack re-syncs an installed dev-linked pack from its recorded source folder.
func (c *Config) DevReloadPack(packID string) (*packs.Manifest, error) {
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return nil, fmt.Errorf("pack_id required")
	}
	c.mu.RLock()
	src := ""
	if c.Packs.DevSources != nil {
		src = strings.TrimSpace(c.Packs.DevSources[packID])
	}
	c.mu.RUnlock()
	if src == "" {
		return nil, fmt.Errorf("pack %q is not dev-linked", packID)
	}
	wasEnabled := c.IsPackEnabled(packID)
	m, err := packs.SyncPackFromDir(src)
	if err != nil {
		return nil, err
	}
	if m.ID != packID {
		return nil, fmt.Errorf("dev source manifest id %q does not match linked pack %q", m.ID, packID)
	}
	if wasEnabled {
		if err := c.SetPackEnabled(packID, false); err != nil {
			return m, err
		}
		if err := c.SetPackEnabled(packID, true); err != nil {
			return m, err
		}
	}
	return m, nil
}

// DevUnlinkPack clears dev_sources for a pack without uninstalling.
func (c *Config) DevUnlinkPack(packID string) error {
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return fmt.Errorf("pack_id required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Packs.DevSources == nil {
		return fmt.Errorf("pack %q is not dev-linked", packID)
	}
	if _, ok := c.Packs.DevSources[packID]; !ok {
		return fmt.Errorf("pack %q is not dev-linked", packID)
	}
	delete(c.Packs.DevSources, packID)
	return nil
}

// DevSourcePath returns the linked dev folder for a pack, if any.
func (c *Config) DevSourcePath(packID string) string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Packs.DevSources == nil {
		return ""
	}
	return strings.TrimSpace(c.Packs.DevSources[packID])
}

// IsPackDevLinked reports whether packID has a dev source folder recorded.
func (c *Config) IsPackDevLinked(packID string) bool {
	return c.DevSourcePath(packID) != ""
}
