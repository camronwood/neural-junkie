package config

import (
	"testing"
)

func TestResolvedSessionDefaultsOff(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_PERSIST_LAST_SESSION", "")
	t.Setenv("NEURAL_JUNKIE_DISABLE_SESSION_PERSIST", "")
	t.Setenv("NEURAL_JUNKIE_RESTORE_LAST_SESSION", "")
	t.Setenv("NEURAL_JUNKIE_SKIP_SESSION_RESTORE", "")
	cfg := DefaultConfig()
	got := cfg.ResolvedSession()
	if got.PersistEnabled || got.RestoreOnStartup {
		t.Fatalf("expected persist/restore off by default, got %+v", got)
	}
}

func TestResolvedSessionPersistEnv(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_PERSIST_LAST_SESSION", "1")
	t.Setenv("NEURAL_JUNKIE_DISABLE_SESSION_PERSIST", "")
	cfg := DefaultConfig()
	got := cfg.ResolvedSession()
	if !got.PersistEnabled {
		t.Fatal("expected PERSIST_LAST_SESSION=1 to enable persist")
	}
	t.Setenv("NEURAL_JUNKIE_DISABLE_SESSION_PERSIST", "1")
	got = cfg.ResolvedSession()
	if got.PersistEnabled {
		t.Fatal("expected DISABLE_SESSION_PERSIST=1 to force persist off")
	}
}

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
