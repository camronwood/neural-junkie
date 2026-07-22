package main

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestSemanticTurnRouterEnabledByDefault(t *testing.T) {
	cfg := &config.Config{Routing: config.DefaultRoutingConfig()}
	router := semanticTurnRouter(cfg)
	if router == nil {
		t.Fatal("semantic router disabled by default")
	}
	rc := cfg.Routing.Normalized()
	if rc.SemanticClassifierModel != config.SemanticClassifierOllamaModel {
		t.Fatalf("semantic model=%q, want %q", rc.SemanticClassifierModel, config.SemanticClassifierOllamaModel)
	}
	if rc.SemanticClassifierTimeoutMS != 8000 {
		t.Fatalf("timeout_ms=%d, want 8000", rc.SemanticClassifierTimeoutMS)
	}
	if router.Timeout != 8*time.Second {
		t.Fatalf("router timeout=%s, want 8s", router.Timeout)
	}
}

func TestSemanticTurnRouterEmergencyRollback(t *testing.T) {
	cfg := &config.Config{Routing: config.DefaultRoutingConfig()}
	cfg.Routing.SemanticRoutingLegacyRollback = true
	if router := semanticTurnRouter(cfg); router != nil {
		t.Fatal("semantic router remained enabled during legacy rollback")
	}
}
