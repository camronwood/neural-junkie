package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestCapabilityPolicyLegacyConfigPreservesSensitiveAccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"server":{},"ai":{},"agents":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.CapabilityPolicy.AllowSensitiveByDefault {
		t.Fatal("legacy config should preserve existing sensitive capability access")
	}
}

func TestResolveAgentCapabilitiesBroadSafeAndSensitiveOptIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackLifeSciences)
	config.InstallTestPack(t, cfg, config.PackAWS)
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackAWS, true); err != nil {
		t.Fatal(err)
	}

	state := cfg.ResolveAgentCapabilities("agent-1", "general", "Helper")
	if !containsCapability(state.Effective, "biology-api") {
		t.Fatalf("safe capability not effective: %+v", state)
	}
	if containsCapability(state.Effective, "aws-api") {
		t.Fatalf("sensitive capability should require opt-in: %+v", state)
	}
	if err := cfg.SetAgentCapabilityOverride("agent-1", config.AgentCapabilityOverride{Allow: []string{"aws-api"}}); err != nil {
		t.Fatal(err)
	}
	state = cfg.ResolveAgentCapabilities("agent-1", "general", "Helper")
	if !containsCapability(state.Effective, "aws-api") {
		t.Fatalf("explicit sensitive grant not effective: %+v", state)
	}
}

func TestResolveAgentCapabilitiesDenyWins(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackLifeSciences)
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetAgentCapabilityOverride("agent-1", config.AgentCapabilityOverride{
		Allow: []string{"biology-api"},
		Deny:  []string{"biology-api"},
	}); err != nil {
		t.Fatal(err)
	}
	state := cfg.ResolveAgentCapabilities("agent-1", "biology", "BiologyExpert")
	if containsCapability(state.Effective, "biology-api") || !containsCapability(state.Denied, "biology-api") {
		t.Fatalf("deny must win: %+v", state)
	}
}

func containsCapability(values []string, id string) bool {
	for _, value := range values {
		if value == id {
			return true
		}
	}
	return false
}
