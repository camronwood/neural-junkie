package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/pipeline"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/trace"
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
		"prepare_turn", "intent_classify", "context_select", "knowledge_plan", "knowledge_execute",
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

type validationLadderRouting struct {
	locals         []ai.AIProvider
	frontier       ai.AIProvider
	allowFrontier  bool
	nextLocal      int
	frontierRouted bool
	calls          int
}

func (r *validationLadderRouting) EffectiveAI(_ context.Context, base ai.AIProvider, _ protocol.AgentInfo, _ *protocol.Message, _ ConversationTrustDecision) ai.AIProvider {
	r.calls++
	if r.nextLocal < len(r.locals) {
		next := r.locals[r.nextLocal]
		r.nextLocal++
		return next
	}
	if r.allowFrontier && !r.frontierRouted && r.frontier != nil {
		r.frontierRouted = true
		return r.frontier
	}
	return base
}

func (p *validationRetryProvider) GenerateResponse(context.Context, string, []protocol.Message) (string, error) {
	p.calls++
	return p.response, nil
}

func newValidationLadderState(t *testing.T, router ChatRouting) (*turnState, *validationRetryProvider, *protocol.Message) {
	t.Helper()
	previous := GlobalChatRouting()
	t.Cleanup(func() { SetGlobalChatRouting(previous) })
	SetGlobalChatRouting(router)

	base := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: "unused base retry"}
	base.Model = "small-local:3b"
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, base, nil)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{ID: "user", Name: "Camron"}, "What does a mutex do?")
	msg.Metadata[protocol.MetadataRoutingAttempts] = []protocol.RoutingAttempt{{
		ProviderID: base.Model, Model: base.Model, Tier: string(ConversationTierStandard), Reason: "standard_route",
	}}
	a.RecordRoutingSnapshot(RoutingSnapshot{
		ConversationTier: string(ConversationTierStandard),
		ChatModel:        base.Model,
		Attempts:         protocol.ExtractRoutingMeta(msg).Attempts,
	})
	responseMsg := protocol.NewMessage(protocol.MessageTypeChat, "dm", a.Info, "The user wants to know what a mutex does.")
	recorder := trace.NewRecorder("turn", "dm", "assistant")
	ctx := trace.WithRecorder(context.Background(), recorder)
	st := &turnState{
		agent: a, ctx: ctx, msg: msg, intent: IntentSubstantive,
		goal: deriveTurnGoal(a, msg, IntentSubstantive),
		context: protocol.TurnContextEnvelope{Corrections: []protocol.TurnContextCorrection{{
			MessageID: "correction-1", Instruction: "Answer the latest mutex question directly.",
		}}},
		evidence: &ActionEvidenceLedger{}, eff: base,
		response: responseMsg.Content, responseMsg: responseMsg, outcome: turnContinue,
		traceRecorder: recorder,
	}
	return st, base, responseMsg
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

func TestValidateResponseCompletesFullLocalLadderSameTurn(t *testing.T) {
	primary := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: ""}
	primary.Model = "primary-local:9b"
	heavy := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: "A mutex protects shared state from concurrent access."}
	heavy.Model = "reliable-local:27b"
	router := &validationLadderRouting{locals: []ai.AIProvider{primary, heavy}}
	st, base, responseMsg := newValidationLadderState(t, router)

	if err := st.stepValidateResponse(st.ctx); err != nil {
		t.Fatal(err)
	}
	if st.response != heavy.response || st.validationAttempts != 2 {
		t.Fatalf("response=%q attempts=%d, want heavy local success on attempt 2", st.response, st.validationAttempts)
	}
	if base.calls != 0 || primary.calls != 1 || heavy.calls != 1 {
		t.Fatalf("calls base=%d primary=%d heavy=%d", base.calls, primary.calls, heavy.calls)
	}
	if !st.contextRecovered {
		t.Fatal("expected one durable-context recovery")
	}
	st.agent.ApplyRoutingMetadataToResponse(responseMsg)
	meta := protocol.ExtractRoutingMeta(responseMsg)
	if len(meta.Attempts) != 3 ||
		meta.Attempts[0].FailureReason == "" ||
		meta.Attempts[1].FailureReason == "" ||
		meta.Attempts[2].FailureReason != "" {
		t.Fatalf("routing attempts=%+v", meta.Attempts)
	}
	snapshot := st.traceRecorder.Snapshot()
	var modelAttempts, contextRecoveries int
	for _, span := range snapshot.Spans {
		switch span.Name {
		case "model_attempt":
			modelAttempts++
		case "context_recovery":
			contextRecoveries++
		}
	}
	if modelAttempts != 2 || contextRecoveries != 1 {
		t.Fatalf("trace model_attempt=%d context_recovery=%d", modelAttempts, contextRecoveries)
	}
}

func TestValidateResponseExhaustsLocalsWithFrontierBlocked(t *testing.T) {
	primary := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: ""}
	primary.Model = "primary-local:9b"
	heavy := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: ""}
	heavy.Model = "reliable-local:27b"
	frontier := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: "A mutex protects shared state."}
	frontier.Model = "frontier-model"
	router := &validationLadderRouting{
		locals: []ai.AIProvider{primary, heavy}, frontier: frontier, allowFrontier: false,
	}
	st, _, responseMsg := newValidationLadderState(t, router)

	if err := st.stepValidateResponse(st.ctx); err != nil {
		t.Fatal(err)
	}
	if st.validationAttempts != 2 || frontier.calls != 0 {
		t.Fatalf("attempts=%d frontier calls=%d", st.validationAttempts, frontier.calls)
	}
	if st.response != "I couldn't produce a sufficiently grounded answer from the available context." {
		t.Fatalf("response=%q", st.response)
	}
	st.agent.ApplyRoutingMetadataToResponse(responseMsg)
	meta := protocol.ExtractRoutingMeta(responseMsg)
	if len(meta.Attempts) != 3 || meta.Attempts[2].FailureReason == "" {
		t.Fatalf("routing attempts=%+v", meta.Attempts)
	}
}

func TestValidateResponseUsesConsentedFrontierAfterLocalExhaustion(t *testing.T) {
	primary := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: ""}
	primary.Model = "primary-local:9b"
	heavy := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: ""}
	heavy.Model = "reliable-local:27b"
	frontier := &validationRetryProvider{MockProvider: ai.NewMockProvider(), response: "A mutex protects shared state from concurrent access."}
	frontier.Model = "frontier-model"
	router := &validationLadderRouting{
		locals: []ai.AIProvider{primary, heavy}, frontier: frontier, allowFrontier: true,
	}
	st, _, responseMsg := newValidationLadderState(t, router)

	if err := st.stepValidateResponse(st.ctx); err != nil {
		t.Fatal(err)
	}
	if st.response != frontier.response || st.validationAttempts != 3 || frontier.calls != 1 {
		t.Fatalf("response=%q attempts=%d frontier calls=%d", st.response, st.validationAttempts, frontier.calls)
	}
	st.agent.ApplyRoutingMetadataToResponse(responseMsg)
	meta := protocol.ExtractRoutingMeta(responseMsg)
	if len(meta.Attempts) != 4 ||
		meta.Attempts[2].FailureReason == "" ||
		meta.Attempts[3].Model != frontier.Model ||
		meta.Attempts[3].FailureReason != "" {
		t.Fatalf("routing attempts=%+v", meta.Attempts)
	}
}

func TestContextSelectionTraceAttrsExposeProvenanceRecoveryAndSizes(t *testing.T) {
	envelope := protocol.TurnContextEnvelope{
		Summary:              &protocol.ConversationSummary{Version: 3, Digest: "durable"},
		Corrections:          []protocol.TurnContextCorrection{{MessageID: "fix-1", Instruction: "use blue"}},
		SupersededMessageIDs: []string{"old-1"},
		Provenance: []protocol.TurnContextProvenance{{
			ID: "msg-2", Section: "recent_exchanges", Source: "channel_history", Score: 0.9, Freshness: "current",
		}},
		SectionBudgets: map[string]int{"recent_exchanges": 8000},
	}
	attrs := contextSelectionTraceAttrs(envelope)
	if attrs["digest_version"] != 3 {
		t.Fatalf("digest version=%v", attrs["digest_version"])
	}
	recovery := attrs["recovery"].(map[string]any)
	if recovery["active"] != true || recovery["correction_count"] != 1 {
		t.Fatalf("recovery=%+v", recovery)
	}
	omissions := attrs["omission_reasons"].(map[string]string)
	if omissions["old-1"] != "superseded" {
		t.Fatalf("omissions=%+v", omissions)
	}
	sizes := attrs["section_sizes"].(map[string]map[string]int)
	if sizes["corrections"]["items"] != 1 || sizes["summary"]["bytes"] == 0 {
		t.Fatalf("sizes=%+v", sizes)
	}
}

func TestStampContextObservabilityCopiesRetrievalCountsAndAnnotatesCompression(t *testing.T) {
	recorder := trace.NewRecorder("turn", "ch", "agent")
	contextSpan := recorder.StartSpan("context_select", nil)
	contextSpan.End(nil)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "user"}, "question")
	msg.Metadata["injected_memory_count"] = 2
	msg.Metadata["injected_codebase_count"] = 4
	msg.Metadata[contextRetrieveCapabilityMetadata] = true
	msg.Metadata[contextBudgetStatsMetadata] = ContextBudgetStats{
		OriginalBytes: 100, FinalBytes: 60, Truncated: true, CompressedSections: []string{"summary"},
	}
	response := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "agent"}, "answer")
	st := &turnState{msg: msg, contextSpan: contextSpan}
	st.stampContextObservability(response)

	if response.Metadata["injected_memory_count"] != 2 || response.Metadata["injected_codebase_count"] != 4 {
		t.Fatalf("response metadata=%+v", response.Metadata)
	}
	attrs := recorder.Snapshot().Spans[0].Attrs
	compression := attrs["compression"].(map[string]any)
	if compression["applied"] != true || compression["recoverable"] != true {
		t.Fatalf("compression=%+v", compression)
	}
}
