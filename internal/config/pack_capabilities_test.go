package config_test

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestResolvedCapabilityRegistry_customerLabPack(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackLifeSciences)
	data, err := buildCustomerLabPackZip(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.InstallPackFromZip(data); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled("customer-lab-pack", true); err != nil {
		t.Fatal(err)
	}
	reg := cfg.ResolvedCapabilityRegistry()
	if !cfg.HasPackCapability("phoenix-import") {
		t.Fatal("expected phoenix-import")
	}
	if !cfg.HasPackCapability("customer-lab-pack/phoenix-import") {
		t.Fatal("expected qualified phoenix-import")
	}
	found := false
	for _, rc := range reg.CapabilityRegistry {
		if rc.ID == "phoenix-import" && rc.Kind == "hub-sidecar" {
			found = true
		}
	}
	if !found {
		t.Fatalf("registry: %+v", reg.CapabilityRegistry)
	}
	if cfg.RouteOwnerPackID("/api/phoenix") != "customer-lab-pack" {
		t.Fatalf("route owner: %q", cfg.RouteOwnerPackID("/api/phoenix"))
	}
}
