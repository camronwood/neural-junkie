package ai

import "testing"

func TestPrependCLIModelArgs(t *testing.T) {
	got := prependCLIModelArgs("claude-cli", []string{"-p"}, "sonnet")
	want := []string{"--model", "sonnet", "-p"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}

	unchanged := prependCLIModelArgs("gemini-cli", []string{"-p"}, "gemini-2.5-flash")
	if len(unchanged) != 1 || unchanged[0] != "-p" {
		t.Fatalf("gemini should not get --model flag, got %v", unchanged)
	}

	already := prependCLIModelArgs("cursor-cli", []string{"--model", "auto", "-p"}, "sonnet")
	wantUnchanged := []string{"--model", "auto", "-p"}
	if len(already) != len(wantUnchanged) {
		t.Fatalf("should not duplicate --model, got %v", already)
	}
	for i := range wantUnchanged {
		if already[i] != wantUnchanged[i] {
			t.Fatalf("should not duplicate --model, got %v", already)
		}
	}
}

func TestEffectiveCLIModel(t *testing.T) {
	p := &CLIAgentProvider{
		ProviderName: "claude-cli",
		Model:        "sonnet",
	}
	if p.EffectiveCLIModel() != "sonnet" {
		t.Fatalf("expected sonnet, got %q", p.EffectiveCLIModel())
	}
	p.Model = "claude-agent"
	if p.EffectiveCLIModel() != "" {
		t.Fatalf("placeholder model should be ignored, got %q", p.EffectiveCLIModel())
	}
	g := &CLIAgentProvider{
		ProviderName: "gemini-cli",
		Env:          map[string]string{"GEMINI_MODEL": "gemini-2.5-pro"},
		Model:        "gemini-agent",
	}
	if g.EffectiveCLIModel() != "gemini-2.5-pro" {
		t.Fatalf("expected gemini env model, got %q", g.EffectiveCLIModel())
	}

	SetCLIProviderModelOverride("cursor-cli", "opus")
	defer SetCLIProviderModelOverride("cursor-cli", "")
	c := &CLIAgentProvider{ProviderName: "cursor-cli", Model: "cursor-agent"}
	if c.EffectiveCLIModel() != "opus" {
		t.Fatalf("expected runtime override opus, got %q", c.EffectiveCLIModel())
	}
}
