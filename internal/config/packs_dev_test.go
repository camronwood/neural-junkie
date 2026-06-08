package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDevLinkPack(t *testing.T) {
	cfg := testConfig(t)
	src := filepath.Join("..", "packs", "testdata", "customer-lab-pack")
	m, err := cfg.DevLinkPack(src)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "customer-lab-pack" {
		t.Fatalf("id=%q", m.ID)
	}
	if !cfg.IsPackDevLinked(m.ID) {
		t.Fatal("expected dev linked")
	}
	if cfg.DevSourcePath(m.ID) == "" {
		t.Fatal("expected dev source path")
	}
	st := cfg.ListPackStatus()
	var found bool
	for _, p := range st.Packs {
		if p.ID == m.ID {
			found = true
			if !p.DevLinked {
				t.Fatal("expected DevLinked in status")
			}
		}
	}
	if !found {
		t.Fatal("pack not in status list")
	}
}

func TestDevReloadPack(t *testing.T) {
	cfg := testConfig(t)
	src := filepath.Join("..", "packs", "testdata", "customer-lab-pack")
	if _, err := cfg.DevLinkPack(src); err != nil {
		t.Fatal(err)
	}
	m, err := cfg.DevReloadPack("customer-lab-pack")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "customer-lab-pack" {
		t.Fatalf("id=%q", m.ID)
	}
}

func TestDevLinkPreservesEnabledState(t *testing.T) {
	cfg := testConfig(t)
	src := filepath.Join("..", "packs", "testdata", "customer-lab-pack")
	if err := cfg.InstallPack(PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	m, err := cfg.DevLinkPack(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(m.ID, true); err != nil {
		t.Fatal(err)
	}
	m2, err := cfg.DevLinkPack(src)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsPackEnabled(m2.ID) {
		t.Fatal("expected dev-link to preserve enabled state")
	}
}

func TestValidatePackDir_requiresPacksStatus(t *testing.T) {
	cfg := testConfig(t)
	src := filepath.Join("..", "packs", "testdata", "customer-lab-pack")
	report, err := cfg.ValidatePackDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.RequiresPacks) == 0 {
		t.Fatal("expected requires_packs status")
	}
	for _, req := range report.RequiresPacks {
		if req.ID == "life-sciences" && req.Installed {
			t.Fatal("life-sciences should not be installed in fresh test config")
		}
	}
}

func testConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	cfg := &Config{
		Packs: PacksConfig{
			Enabled: make(map[string]bool),
		},
	}
	_ = os.Setenv("NEURAL_JUNKIE_DATA_DIR", dir)
	t.Cleanup(func() { _ = os.Unsetenv("NEURAL_JUNKIE_DATA_DIR") })
	return cfg
}
