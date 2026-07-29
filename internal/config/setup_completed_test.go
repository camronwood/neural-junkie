package config

import (
	"encoding/json"
	"testing"
)

func TestMigrateSetupCompleted_legacyTrue(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SetupCompleted {
		t.Fatal("DefaultConfig should start with setup_completed=false")
	}
	cfg.migrateSetupCompleted([]byte(`{"server":{"host":"localhost"}}`))
	if !cfg.SetupCompleted {
		t.Fatal("legacy config without setup_completed should migrate to true")
	}
}

func TestMigrateSetupCompleted_respectsExplicitFalse(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetupCompleted = true
	raw, err := json.Marshal(map[string]any{"setup_completed": false})
	if err != nil {
		t.Fatal(err)
	}
	// Unmarshal first like Load does, then migrate.
	if err := json.Unmarshal(raw, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.migrateSetupCompleted(raw)
	if cfg.SetupCompleted {
		t.Fatal("explicit setup_completed=false must not be overwritten")
	}
}

func TestNeedsSetup(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.NeedsSetup() {
		t.Fatal("expected NeedsSetup for unfinished default config")
	}
	cfg.SetupCompleted = true
	if cfg.NeedsSetup() {
		t.Fatal("expected NeedsSetup=false after setup")
	}
}
