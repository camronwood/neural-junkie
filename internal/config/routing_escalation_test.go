package config

import "testing"

func TestRoutingEscalationDefaultsLocalOnly(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Routing.LocalEscalationEnabled {
		t.Fatal("local escalation must default to enabled")
	}
	if cfg.Routing.FrontierEscalationEnabled {
		t.Fatal("frontier escalation must require explicit consent")
	}
}
