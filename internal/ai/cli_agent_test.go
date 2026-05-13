package ai

import "testing"

func TestSanitizeGeminiCLIPromptEcho(t *testing.T) {
	p := &CLIAgentProvider{ProviderName: "gemini-cli"}
	fp := "Here is the recent conversation context:\n\n[Human User]: hi\n\n---\n\nNow respond to the following:\n\nYou are Gemini."

	t.Run("strips credentials and prompt echo", func(t *testing.T) {
		raw := "Loaded cached credentials.\n" + fp + "\n\nYes, I can help."
		got := p.sanitizeGeminiCLIPromptEcho(raw, fp)
		if got != "Yes, I can help." {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("non-gemini provider unchanged", func(t *testing.T) {
		other := &CLIAgentProvider{ProviderName: "cursor-cli"}
		raw := "Loaded cached credentials.\n" + fp + "\nbody"
		got := other.sanitizeGeminiCLIPromptEcho(raw, fp)
		if got != raw {
			t.Fatalf("expected raw unchanged, got %q", got)
		}
	})

	t.Run("double echo uses last prompt", func(t *testing.T) {
		raw := fp + "\n\n" + fp + "\n\nFinal answer only."
		got := p.sanitizeGeminiCLIPromptEcho(raw, fp)
		if got != "Final answer only." {
			t.Fatalf("got %q", got)
		}
	})
}

func TestStripCLILeadingNoiseLines(t *testing.T) {
	s := "Loaded cached credentials.\n\ntype.googleapis.com/google.rpc\n\nHello"
	got := stripCLILeadingNoiseLines(s)
	if got != "Hello" {
		t.Fatalf("got %q", got)
	}
}

func TestShouldUsePTYStreaming_GeminiDefaultOff(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_DISABLE_CLI_PTY", "")
	t.Setenv("NEURAL_JUNKIE_GEMINI_CLI_PTY", "")
	g := &CLIAgentProvider{ProviderName: "gemini-cli"}
	if g.shouldUsePTYStreaming() {
		t.Fatal("expected PTY off for gemini-cli by default")
	}
	t.Setenv("NEURAL_JUNKIE_GEMINI_CLI_PTY", "1")
	if !g.shouldUsePTYStreaming() {
		t.Fatal("expected PTY on when NEURAL_JUNKIE_GEMINI_CLI_PTY=1")
	}
}
