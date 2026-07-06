package ai

import (
	"testing"
)

func TestAppendClaudeCLIEnv_noOpWhenCustomRouting(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING", "1")
	in := []string{"ANTHROPIC_BASE_URL=http://localhost:4000", "PATH=/bin"}
	got := appendClaudeCLIEnv(in)
	if len(got) != len(in) {
		t.Fatalf("expected unchanged env, got %v", got)
	}
}

func TestAppendClaudeCLIEnv_alwaysStripsProxyOverrides(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING", "")
	in := []string{
		"PATH=/bin",
		"ANTHROPIC_BASE_URL=http://localhost:4000",
		"ANTHROPIC_AUTH_TOKEN=sk-1234",
		"ANTHROPIC_API_KEY=sk-ant-api03-test",
		"ANTHROPIC_MODEL=openai/qwen3-coder-30b",
	}
	got := appendClaudeCLIEnv(in)
	if len(got) != 1 {
		t.Fatalf("got %v, want PATH only", got)
	}
	if got[0] != "PATH=/bin" {
		t.Fatalf("unexpected env: %v", got)
	}
}

func TestAppendClaudeCLIEnv_respectsCustomRoutingFlag(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING", "1")
	in := []string{"ANTHROPIC_BASE_URL=http://localhost:4000"}
	got := appendClaudeCLIEnv(in)
	if len(got) != 1 || got[0] != in[0] {
		t.Fatalf("expected unchanged env, got %v", got)
	}
}
