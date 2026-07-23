package ai

import (
	"context"
	"errors"
	"os/exec"
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

func TestFilterBenignCLIStderr(t *testing.T) {
	t.Parallel()
	warn := "[WARN] Skipping unreadable directory: /Library/Bluetooth (EPERM: operation not permitted, scandir '/Library/Bluetooth')"
	if got := filterBenignCLIStderr(warn); got != "" {
		t.Fatalf("expected benign warn filtered, got %q", got)
	}
	mixed := warn + "\nERROR: authentication failed\n"
	if got := filterBenignCLIStderr(mixed); !strings.Contains(got, "authentication failed") {
		t.Fatalf("expected real error kept, got %q", got)
	}
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

func TestCursorTrustArgs_defaultYolo(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_CURSOR_TRUST", "")
	p := &CLIAgentProvider{ProviderName: "cursor-cli", ApprovalMode: "yolo"}
	got := p.cursorTrustArgs()
	if len(got) != 1 || got[0] != "--trust" {
		t.Fatalf("cursorTrustArgs() = %v, want [--trust]", got)
	}
}

func TestCursorTrustArgs_interactiveSkipsTrust(t *testing.T) {
	p := &CLIAgentProvider{ProviderName: "cursor-cli", ApprovalMode: "interactive"}
	if len(p.cursorTrustArgs()) != 0 {
		t.Fatalf("expected no trust args in interactive mode, got %v", p.cursorTrustArgs())
	}
}

func TestBuildCLIInvocationArgs_prependsTrustForCursor(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_CURSOR_TRUST", "")
	p := &CLIAgentProvider{
		ProviderName: "cursor-cli",
		ApprovalMode: "yolo",
		BaseArgs:     []string{"-p", "--output-format", "text"},
	}
	got := p.buildCLIInvocationArgs(p.BaseArgs)
	wantPrefix := []string{"--trust", "-p", "--output-format", "text"}
	if len(got) != len(wantPrefix) {
		t.Fatalf("args = %v, want %v", got, wantPrefix)
	}
	for i := range wantPrefix {
		if got[i] != wantPrefix[i] {
			t.Fatalf("args[%d] = %q, want %q (full: %v)", i, got[i], wantPrefix[i], got)
		}
	}
}

func TestClaudePermissionArgs_autoEdit(t *testing.T) {
	p := &CLIAgentProvider{ProviderName: "claude-cli", ApprovalMode: "auto_edit"}
	got := p.claudePermissionArgs()
	want := []string{"--permission-mode", "acceptEdits"}
	if len(got) != len(want) {
		t.Fatalf("claudePermissionArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClaudePermissionArgs_yolo(t *testing.T) {
	p := &CLIAgentProvider{ProviderName: "claude-cli", ApprovalMode: "yolo"}
	got := p.claudePermissionArgs()
	if len(got) != 2 || got[0] != "--permission-mode" || got[1] != "bypassPermissions" {
		t.Fatalf("claudePermissionArgs() = %v", got)
	}
}

func TestShouldUsePTYStreaming_ClaudeOff(t *testing.T) {
	p := &CLIAgentProvider{ProviderName: "claude-cli"}
	if p.shouldUsePTYStreaming() {
		t.Fatal("expected claude-cli to use pipe streaming, not PTY")
	}
}

func TestShouldUsePTYStreaming_CodexOff(t *testing.T) {
	p := &CLIAgentProvider{ProviderName: "codex-cli"}
	if p.shouldUsePTYStreaming() {
		t.Fatal("expected codex-cli to use pipe streaming, not PTY")
	}
}

func TestAttachClosedCLIStdin(t *testing.T) {
	cmd := exec.Command("true")
	cleanup := attachClosedCLIStdin(cmd)
	defer cleanup()
	if cmd.Stdin == nil {
		t.Fatal("expected Stdin to be set to /dev/null")
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
