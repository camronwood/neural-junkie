package agent

import (
	"os"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestIsDeliverableJudgeMessage(t *testing.T) {
	if isDeliverableJudgeMessage(nil) {
		t.Fatal("nil message")
	}
	if isDeliverableJudgeMessage(&protocol.Message{}) {
		t.Fatal("empty metadata")
	}
	msg := &protocol.Message{Metadata: map[string]interface{}{"deliverable_judge": true}}
	if !isDeliverableJudgeMessage(msg) {
		t.Fatal("expected true for bool metadata")
	}
	msg.Metadata["deliverable_judge"] = "true"
	if !isDeliverableJudgeMessage(msg) {
		t.Fatal("expected true for string metadata")
	}
}

func TestPrepareCLIInvocationJudgeModelOverride(t *testing.T) {
	t.Setenv("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", "gemini-2.5-pro")
	provider := ai.NewCLIAgentProvider("gemini", "/tmp", "gemini-cli",
		ai.WithEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		ai.WithModel("gemini-2.5-flash"),
	)
	ai.SetCLIProviderModelOverride("gemini-cli", "gemini-2.5-flash")
	defer ai.SetCLIProviderModelOverride("gemini-cli", "")

	ag := &Agent{
		Info: protocol.AgentInfo{
			Type:       protocol.AgentTypeCLI,
			AIProvider: "gemini-cli",
		},
		AI: provider,
	}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{"deliverable_judge": true},
	}

	restore := ag.prepareCLIInvocation(msg)
	if provider.Env["GEMINI_MODEL"] != "gemini-2.5-pro" {
		t.Fatalf("expected judge model env, got %q", provider.Env["GEMINI_MODEL"])
	}
	if provider.EffectiveCLIModel() != "gemini-2.5-pro" {
		t.Fatalf("expected judge override, got %q", provider.EffectiveCLIModel())
	}
	restore()
	if provider.Env["GEMINI_MODEL"] != "gemini-2.5-flash" {
		t.Fatalf("expected flash restored, got %q", provider.Env["GEMINI_MODEL"])
	}
	if provider.EffectiveCLIModel() != "gemini-2.5-flash" {
		t.Fatalf("expected flash override restored, got %q", provider.EffectiveCLIModel())
	}
}

func TestResolveDeliverableJudgeGeminiModel(t *testing.T) {
	os.Unsetenv("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL")
	if resolveDeliverableJudgeGeminiModel() != "" {
		t.Fatal("expected empty when env unset")
	}
	t.Setenv("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL", "gemini-2.5-pro")
	if resolveDeliverableJudgeGeminiModel() != "gemini-2.5-pro" {
		t.Fatalf("unexpected model %q", resolveDeliverableJudgeGeminiModel())
	}
}
