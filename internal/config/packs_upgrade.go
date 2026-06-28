package config

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// PackUpdateInfo describes an installed catalog pack with a newer store version.
type PackUpdateInfo struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	InstalledVersion string `json:"installed_version"`
	LatestVersion    string `json:"latest_version"`
	Enabled          bool   `json:"enabled"`
}

// installedCatalogVersion returns the version from the installed pack manifest.
func (c *Config) installedCatalogVersion(packID string) string {
	m, err := c.InstalledPackManifestByID(packID)
	if err != nil || m == nil {
		return ""
	}
	return strings.TrimSpace(m.Version)
}

// packUpdateInfoForEntry returns update metadata for one catalog entry, or nil if up to date / not applicable.
func (c *Config) packUpdateInfoForEntry(e packs.CatalogEntry) *PackUpdateInfo {
	if c == nil || !c.IsPackInstalled(e.ID) {
		return nil
	}
	if c.DevSourcePath(e.ID) != "" {
		return nil
	}
	if m, _ := c.InstalledPackManifestByID(e.ID); m != nil && m.IsCustomerPack() {
		return nil
	}
	installed := c.installedCatalogVersion(e.ID)
	latest := strings.TrimSpace(e.Version)
	if !packs.UpdateAvailable(installed, latest) {
		return nil
	}
	return &PackUpdateInfo{
		ID:               e.ID,
		Title:            e.Title,
		InstalledVersion: installed,
		LatestVersion:    latest,
		Enabled:          c.IsPackEnabled(e.ID),
	}
}

// ListPackUpdates returns installed catalog packs that have a newer version in the store.
func (c *Config) ListPackUpdates() ([]PackUpdateInfo, error) {
	cat, err := packs.FetchCatalog()
	if err != nil {
		return nil, err
	}
	var out []PackUpdateInfo
	for _, e := range cat.Packs {
		if info := c.packUpdateInfoForEntry(e); info != nil {
			out = append(out, *info)
		}
	}
	return out, nil
}

// UpgradePack re-downloads a catalog pack when the store version is newer.
func (c *Config) UpgradePack(packID string) (wasEnabled bool, err error) {
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return false, fmt.Errorf("pack id required")
	}
	if !c.IsPackInstalled(packID) {
		return false, fmt.Errorf("pack %q is not installed", packID)
	}
	if c.DevSourcePath(packID) != "" {
		return false, fmt.Errorf("pack %q is dev-linked — use dev reload instead of upgrade", packID)
	}
	cat, err := packs.FetchCatalog()
	if err != nil {
		return false, err
	}
	e := cat.CatalogEntryByID(packID)
	if e == nil {
		return false, fmt.Errorf("unknown pack %q", packID)
	}
	installed := c.installedCatalogVersion(packID)
	latest := strings.TrimSpace(e.Version)
	if !packs.UpdateAvailable(installed, latest) {
		return false, fmt.Errorf("pack %q is already at version %s (catalog %s)", packID, installed, latest)
	}
	wasEnabled = c.IsPackEnabled(packID)
	if err := packs.InstallOfficialPack(packID); err != nil {
		return wasEnabled, err
	}
	c.mu.Lock()
	if !c.packInstalledLocked(packID) {
		c.Packs.Installed = append(c.Packs.Installed, packID)
	}
	c.mu.Unlock()
	if wasEnabled {
		if err := c.SetPackEnabled(packID, true); err != nil {
			return wasEnabled, err
		}
	} else {
		c.SyncAgentsFromPacks()
	}
	return wasEnabled, nil
}
