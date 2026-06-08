package config

import "testing"

func TestConversationMemoryEnabled_defaultTrue(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.ConversationMemoryEnabled() {
		t.Fatal("expected default on")
	}
}

func TestConversationMemoryEnabled_explicitFalse(t *testing.T) {
	cfg := DefaultConfig()
	f := false
	cfg.Features.ConversationMemoryEnabled = &f
	if cfg.ConversationMemoryEnabled() {
		t.Fatal("expected off when explicitly false")
	}
}
