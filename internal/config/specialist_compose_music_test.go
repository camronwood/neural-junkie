package config

import "testing"

func TestChatModelForAgentMusicPackDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs.Installed = append(cfg.Packs.Installed, PackMusicCreation)
	cfg.Packs.Enabled = map[string]bool{PackMusicCreation: true}
	cfg.SpecialistCompose = map[string]SpecialistComposeEntry{
		"music": {ChatModel: "qwen2.5:7b"},
	}

	if got := cfg.ChatModelForAgent("music", ""); got != "qwen2.5:7b" {
		t.Fatalf("got %q", got)
	}
}

func TestChatModelForAgentMusicFallbackWhenComposeEmpty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs.Installed = append(cfg.Packs.Installed, PackMusicCreation)
	cfg.Packs.Enabled = map[string]bool{PackMusicCreation: true}

	if got := cfg.ChatModelForAgent("music", ""); got != "qwen2.5:7b" {
		t.Fatalf("fallback = %q", got)
	}
}
