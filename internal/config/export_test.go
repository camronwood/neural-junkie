package config

import "testing"

// SetupTestOfficialPackCatalog configures a local catalog and HTTPS zip server for tests.
func SetupTestOfficialPackCatalog(t *testing.T) {
	setupTestOfficialPackCatalog(t)
}

// InstallTestPack syncs an official pack fixture into the test home and updates config.
func InstallTestPack(t *testing.T, cfg *Config, packID string) {
	installTestPack(t, cfg, packID)
}
