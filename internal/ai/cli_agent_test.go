package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

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

func TestTruncateCLIPrompt_KeepsTailTask(t *testing.T) {
	marker := "\n\n---\n\nNow respond to the following:\n\n"
	head := strings.Repeat("a", 30000)
	tail := "Complete the assigned collaboration task now."
	prompt := head + marker + tail
	got := truncateCLIPrompt(prompt, 8000)
	if !strings.Contains(got, tail) {
		t.Fatalf("expected tail preserved, got len=%d suffix=%q", len(got), got[max(0, len(got)-120):])
	}
	if !strings.Contains(got, "prompt truncated") {
		t.Fatalf("expected truncation marker, got len=%d", len(got))
	}
}

func TestNewGeminiCLIProvider_DefaultTimeoutCapped(t *testing.T) {
	p := NewGeminiCLIProvider(".")
	if p.Timeout != DefaultGeminiCLITimeout {
		t.Fatalf("timeout = %v, want %v", p.Timeout, DefaultGeminiCLITimeout)
	}
	p2 := NewGeminiCLIProvider(".", WithTimeout(45*time.Second))
	if p2.Timeout != 45*time.Second {
		t.Fatalf("custom timeout = %v", p2.Timeout)
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

func TestGeminiPipeStreamTimeoutDoesNotHang(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_DISABLE_CLI_PTY", "1")
	p := &CLIAgentProvider{
		Command:      "sh",
		BaseArgs:     []string{"-c", "sleep 10 & wait", "_"},
		Timeout:      50 * time.Millisecond,
		Env:          map[string]string{},
		ProviderName: "gemini-cli",
	}

	tokenCh, err := p.GenerateResponseStream(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("GenerateResponseStream: %v", err)
	}

	select {
	case token, ok := <-tokenCh:
		if !ok {
			t.Fatal("stream closed without timeout token")
		}
		if !errors.Is(token.Error, ErrCLIProviderTimeout) {
			t.Fatalf("expected timeout error, got %#v", token.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for CLI stream timeout")
	}
}
