package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestBuildDiagnoseChecksTokensFail(t *testing.T) {
	out := DiagnoseResult{AppTokenOK: false, BotTokenOK: false}
	checks := buildDiagnoseChecks(out)
	if len(checks) == 0 {
		t.Fatal("expected checks")
	}
	if checks[0].ID != "tokens" || checks[0].Status != "fail" {
		t.Fatalf("tokens check: %+v", checks[0])
	}
}

func TestBuildDiagnoseChecksChannelsWarn(t *testing.T) {
	out := DiagnoseResult{
		AppTokenOK:   true,
		BotTokenOK:   true,
		AuthTestOK:   true,
		SocketOpenOK: true,
		ChannelsFound: 0,
		BindingsCount: 0,
	}
	checks := buildDiagnoseChecks(out)
	var channelsWarn, bindingWarn bool
	for _, c := range checks {
		if c.ID == "channels_found" && c.Status == "warn" {
			channelsWarn = true
		}
		if c.ID == "binding_exists" && c.Status == "warn" {
			bindingWarn = true
		}
	}
	if !channelsWarn || !bindingWarn {
		t.Fatalf("expected channel and binding warns, got %+v", checks)
	}
}

func TestDiagnoseWithRuntimeBindingsCount(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Slack.Enabled = false
	out := DiagnoseWithRuntime(cfg, DiagnoseRuntimeContext{
		BridgeConnected: true,
		BindingsCount:   2,
	})
	if out.BindingsCount != 2 {
		t.Fatalf("bindings count: %d", out.BindingsCount)
	}
	if len(out.Checks) == 0 {
		t.Fatal("expected checks")
	}
}
