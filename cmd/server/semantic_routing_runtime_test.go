package main

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestSemanticTurnRouterEnabledByDefault(t *testing.T) {
	cfg := &config.Config{Routing: config.DefaultRoutingConfig()}
	if router := semanticTurnRouter(cfg); router == nil {
		t.Fatal("semantic router disabled by default")
	}
}

func TestSemanticTurnRouterEmergencyRollback(t *testing.T) {
	cfg := &config.Config{Routing: config.DefaultRoutingConfig()}
	cfg.Routing.SemanticRoutingLegacyRollback = true
	if router := semanticTurnRouter(cfg); router != nil {
		t.Fatal("semantic router remained enabled during legacy rollback")
	}
}
