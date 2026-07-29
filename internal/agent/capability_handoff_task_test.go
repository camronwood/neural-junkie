package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestValidateCapabilityHandoffTask_rejectsMenusAndGreetings(t *testing.T) {
	t.Parallel()
	for _, task := range []string{
		"whats up every body?",
		"debugging a failing pod or reviewing your CI/CD pipeline security",
		"data pipelines, model serving, or any other ML tasks",
		"reviewing API designs, analyzing concurrency patterns, or optimizing service architectures",
		"Assist with data pipelines, model serving, or other ML tasks",
		"explain",
		"assist_user",
		"help",
	} {
		if err := ValidateCapabilityHandoffTask(task); err == nil {
			t.Fatalf("expected rejection for %q", task)
		}
	}
}

func TestValidateCapabilityHandoffTask_allowsConcreteTasks(t *testing.T) {
	t.Parallel()
	for _, task := range []string{
		"Verify the page accessibility",
		"answer: What is 2+2?",
		"Inspect pods in namespace payments for CrashLoopBackOff and summarize the last 50 log lines",
		"Review auth.go for a race around session refresh",
	} {
		if err := ValidateCapabilityHandoffTask(task); err != nil {
			t.Fatalf("expected allow for %q: %v", task, err)
		}
	}
}

func TestCapabilityHandoffTurnState_oncePerTurn(t *testing.T) {
	t.Parallel()
	ctx := withCapabilityHandoffTurnState(context.Background())
	st := capabilityHandoffTurnStateFromContext(ctx)
	if st == nil {
		t.Fatal("expected turn state")
	}
	st.count = 1

	a := &Agent{Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := &protocol.Message{ID: "m1", Channel: "general", From: protocol.AgentInfo{Name: "camron"}, Content: "debug the failing auth middleware in auth.go"}
	out, err := a.executeRequestCapabilityHelpTool(ctx, msg, []byte(`{"capability_id":"web-browser","task":"Verify the page accessibility"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "already used this turn") {
		t.Fatalf("expected once-per-turn refusal, got %q", out)
	}
}

func TestExecuteRequestCapabilityHelpTool_rejectsVagueTaskWithoutHandoff(t *testing.T) {
	t.Parallel()
	ctx := withCapabilityHandoffTurnState(context.Background())
	a := &Agent{Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := &protocol.Message{ID: "m1", Channel: "general", From: protocol.AgentInfo{Name: "camron"}, Content: "debug the failing auth middleware in auth.go"}
	out, err := a.executeRequestCapabilityHelpTool(ctx, msg, []byte(`{"capability_id":"web-browser","task":"whats up every body?"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "greeting") && !strings.Contains(strings.ToLower(out), "not a bounded") {
		t.Fatalf("expected vague-task refusal, got %q", out)
	}
	if st := capabilityHandoffTurnStateFromContext(ctx); st == nil || st.count != 0 {
		t.Fatal("vague rejection must not consume the per-turn budget")
	}
}

func TestShouldOfferCapabilityTools_answerWithoutWorkSignals(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "what should we talk about today?"}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if shouldOfferCapabilityTools(msg) {
		t.Fatal("expected capability tools suppressed for answer without work signals")
	}
}

func TestShouldOfferCapabilityTools_casualInteraction(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "lets brainstorm devops ideas"}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionCasual,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if shouldOfferCapabilityTools(msg) {
		t.Fatal("expected capability tools suppressed for casual interaction")
	}
}

func TestIsSocialOrStatusPing_whatsUpEverybody(t *testing.T) {
	t.Parallel()
	if !isSocialOrStatusPing("whats up every body?") {
		t.Fatal("expected social ping")
	}
	if !isSocialOrStatusPing("what's up everyone") {
		t.Fatal("expected social ping")
	}
}
