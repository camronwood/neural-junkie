package config

import (
	"testing"
)

func TestResolvedSecurityEnvOverridesConfig(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	cfg := DefaultConfig()
	cfg.Security.AuthRequired = false
	got := cfg.ResolvedSecurity()
	if !got.AuthRequired {
		t.Fatal("expected env to override auth_required")
	}
}

func TestResolvedImageGenFromConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ImageGen.Provider = "openai"
	cfg.ImageGen.Model = "dall-e-3"
	got := cfg.ResolvedImageGen()
	if got.Provider != "openai" || got.Model != "dall-e-3" {
		t.Fatalf("got %+v", got)
	}
}

func TestSeedFromEnvAuth(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	cfg := DefaultConfig()
	cfg.SeedFromEnv()
	if !cfg.Security.AuthRequired {
		t.Fatal("expected seed to set auth_required")
	}
}

func TestResolvedCLIAgentsFromConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CLIAgents.DisablePTY = true
	cfg.CLIAgents.CursorTrust = false
	got := cfg.ResolvedCLIAgents()
	if !got.DisablePTY || got.CursorTrust {
		t.Fatalf("got %+v", got)
	}
}

func TestSettingsRestartReasonsListenAll(t *testing.T) {
	prev := DefaultConfig().CaptureSettingsRestartBaseline()
	next := DefaultConfig()
	next.Server.ListenAll = true
	reasons := SettingsRestartReasons(prev, next)
	found := false
	for _, r := range reasons {
		if r == "server.listen_all" || r == "listen_all" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected listen_all in %+v", reasons)
	}
}
