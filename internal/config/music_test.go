package config

import "testing"

func TestMusicMCPConfigNormalizedDefaults(t *testing.T) {
	n := MusicMCPConfig{ModelVariant: "turbo"}.Normalized()
	if n.InferenceSteps != 8 || n.GuidanceScale != 1.0 {
		t.Fatalf("turbo defaults = %+v", n)
	}
	if n.InferMethod != "ode" {
		t.Fatalf("infer method = %q", n.InferMethod)
	}
}

func TestMusicSidecarSettingsIncludesCheckpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP.Music = MusicMCPConfig{ModelVariant: "xl-sft", InferenceSteps: 50, GuidanceScale: 7, InferMethod: "ode"}
	settings := cfg.MusicSidecarSettings()
	if settings["ace_step_model_variant"] != "xl-sft" {
		t.Fatalf("variant = %q", settings["ace_step_model_variant"])
	}
	if settings["ace_step_checkpoint"] == "" {
		t.Fatal("expected checkpoint path")
	}
}
