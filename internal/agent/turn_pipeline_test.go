package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/pipeline"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestDefaultTurnPipelineStepOrder(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "Test", ID: "a1"}}
	st := &turnState{
		agent:   a,
		msg:     protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "hello"),
		outcome: turnContinue,
	}
	steps := a.defaultTurnPipeline(st)
	var names []string
	for _, s := range steps {
		names = append(names, s.Name())
	}
	want := []string{
		"prepare_turn", "intent_classify", "knowledge_plan", "knowledge_execute",
		"governance_record", "provider_route", "generate", "post_process",
		"validate_response", "stamp_metadata", "deliver_response",
	}
	if len(names) != len(want) {
		t.Fatalf("steps=%v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("step[%d]=%q want %q", i, names[i], want[i])
		}
	}
}

func TestPipelineRunWithHooks(t *testing.T) {
	var ran []string
	steps := []pipeline.Step{
		pipeline.FuncStep{StepName: "intent_classify", Fn: func(ctx context.Context) error {
			ran = append(ran, "intent_classify")
			return nil
		}},
	}
	if err := pipeline.RunWithHooks(context.Background(), steps, func(n string) { ran = append(ran, "before:"+n) }, func(n string) { ran = append(ran, "after:"+n) }); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 3 {
		t.Fatalf("ran=%v", ran)
	}
}

type validationRetryProvider struct {
	*ai.MockProvider
	calls    int
	response string
}

func (p *validationRetryProvider) GenerateResponse(context.Context, string, []protocol.Message) (string, error) {
	p.calls++
	return p.response, nil
}

func TestValidateResponseRetryUsesReliableProviderOnce(t *testing.T) {
	previous := GlobalChatRouting()
	t.Cleanup(func() { SetGlobalChatRouting(previous) })

	base := &validationRetryProvider{
		MockProvider: ai.NewMockProvider(),
		response:     "base response must not be used",
	}
	base.Model = "standard-model"
	reliable := &validationRetryProvider{
		MockProvider: ai.NewMockProvider(),
		response:     "A mutex protects shared state from concurrent access.",
	}
	reliable.Model = "reliable-model"
	router := &trustRoutingStub{next: reliable}
	SetGlobalChatRouting(router)

	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, base, nil)
	a.RecordRoutingSnapshot(RoutingSnapshot{ConversationTier: string(ConversationTierElevated), ChatModel: base.Model})
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{ID: "user", Name: "Camron"}, "What does a mutex do?")
	responseMsg := protocol.NewMessage(protocol.MessageTypeChat, "dm", a.Info, "The user wants to know what a mutex does.")
	st := &turnState{
		agent:       a,
		ctx:         context.Background(),
		msg:         msg,
		intent:      IntentSubstantive,
		goal:        deriveTurnGoal(a, msg, IntentSubstantive),
		evidence:    &ActionEvidenceLedger{},
		eff:         base,
		response:    responseMsg.Content,
		responseMsg: responseMsg,
		outcome:     turnContinue,
	}

	if err := st.stepValidateResponse(context.Background()); err != nil {
		t.Fatal(err)
	}
	if router.calls != 1 {
		t.Fatalf("trust router calls=%d want 1", router.calls)
	}
	if reliable.calls != 1 {
		t.Fatalf("reliable provider calls=%d want 1", reliable.calls)
	}
	if base.calls != 0 {
		t.Fatalf("standard provider retry calls=%d want 0", base.calls)
	}
	if !st.validationRetried || st.eff != reliable {
		t.Fatalf("retry state=%v provider=%T", st.validationRetried, st.eff)
	}
	if st.response != reliable.response {
		t.Fatalf("response=%q want %q", st.response, reliable.response)
	}

	a.ApplyRoutingMetadataToResponse(responseMsg)
	meta := protocol.ExtractRoutingMeta(responseMsg)
	if meta.Model != "reliable-model" || meta.ConversationTier != string(ConversationTierReliable) {
		t.Fatalf("routing metadata=%+v", meta)
	}
	if !conversationContainsString(meta.ConversationReasons, ConversationReasonQualityGateFailure) {
		t.Fatalf("routing reasons=%v", meta.ConversationReasons)
	}
}
