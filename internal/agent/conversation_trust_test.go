package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type trustRoutingStub struct {
	calls int
	got   ConversationTrustDecision
	next  ai.AIProvider
}

func (s *trustRoutingStub) EffectiveAI(_ context.Context, base ai.AIProvider, _ protocol.AgentInfo, _ *protocol.Message, trust ConversationTrustDecision) ai.AIProvider {
	s.calls++
	s.got = trust
	if s.next != nil {
		return s.next
	}
	return base
}

func TestClassifyConversationTrustSignals(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	prior := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{ID: "user", Name: "Camron", Type: protocol.AgentTypeGeneral}, "Explain the deployment failure")
	a.replaceChannelHistory("dm", []*protocol.Message{prior})

	tests := []struct {
		name    string
		content string
		meta    map[string]interface{}
		want    ConversationTier
		reason  string
	}{
		{name: "standard", content: "What is a mutex?", want: ConversationTierStandard},
		{name: "explicit tool action", content: "Run the focused tests", want: ConversationTierElevated, reason: ConversationReasonExplicitToolAction},
		{name: "large context", content: strings.Repeat("x", conversationLargeContextBytes), want: ConversationTierElevated, reason: ConversationReasonLargeContext},
		{
			name: "full workspace context", content: "hello",
			meta: map[string]interface{}{
				MetadataContextScope: ContextScopeFull,
				"workspace_context": map[string]interface{}{
					"workspace_path": "/project",
					"open_files": []interface{}{
						map[string]interface{}{"path": "main.go", "content": "package main"},
					},
				},
			},
			want: ConversationTierElevated, reason: ConversationReasonLargeContext,
		},
		{
			name: "large focused workspace payload", content: "hello",
			meta: map[string]interface{}{
				MetadataContextScope: ContextScopeFocus,
				"workspace_context": map[string]interface{}{
					"open_files": []interface{}{
						map[string]interface{}{"path": "large.go", "content": strings.Repeat("x", conversationLargeContextBytes)},
					},
				},
			},
			want: ConversationTierElevated, reason: ConversationReasonLargeContext,
		},
		{
			name: "small none workspace context", content: "hello",
			meta: map[string]interface{}{
				MetadataContextScope: ContextScopeNone,
				"workspace_context": map[string]interface{}{
					"workspace_path": "/project",
					"open_files":     []interface{}{},
				},
			},
			want: ConversationTierStandard,
		},
		{
			name: "small focus workspace context", content: "hello",
			meta: map[string]interface{}{
				MetadataContextScope: ContextScopeFocus,
				"workspace_context": map[string]interface{}{
					"file_tree": "main.go",
					"open_files": []interface{}{
						map[string]interface{}{"path": "main.go", "content": "package main"},
					},
				},
			},
			want: ConversationTierStandard,
		},
		{name: "correction", content: "No, that is wrong; use the configured provider", want: ConversationTierReliable, reason: ConversationReasonUserCorrection},
		{name: "repeated from history", content: "Explain the deployment failure", want: ConversationTierReliable, reason: ConversationReasonRepeatedRequest},
		{name: "quality failure", content: "answer", meta: map[string]interface{}{conversationQualityFailureKey: true}, want: ConversationTierReliable, reason: ConversationReasonQualityGateFailure},
		{name: "mutation trust is independent", content: "hello", meta: map[string]interface{}{"mutation_trust": "elevated"}, want: ConversationTierStandard},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{ID: "user", Name: "Camron"}, tc.content)
			msg.Metadata = tc.meta
			got := a.ClassifyConversationTrust(msg)
			if got.Tier != tc.want {
				t.Fatalf("tier = %q, want %q (reasons=%v)", got.Tier, tc.want, got.Reasons)
			}
			if tc.reason != "" && !conversationContainsString(got.Reasons, tc.reason) {
				t.Fatalf("reasons = %v, want %q", got.Reasons, tc.reason)
			}
		})
	}
}

func TestEscalateConversationProviderAllowsOneReliableReroute(t *testing.T) {
	previous := GlobalChatRouting()
	t.Cleanup(func() { SetGlobalChatRouting(previous) })

	reliable := ai.NewMockProvider()
	reliable.Model = "reliable-model"
	router := &trustRoutingStub{next: reliable}
	SetGlobalChatRouting(router)

	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	a.RecordRoutingSnapshot(RoutingSnapshot{ConversationTier: string(ConversationTierElevated)})
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{ID: "user", Name: "Camron"}, "Please fix the answer")

	got, ok := a.EscalateConversationProvider(context.Background(), msg)
	if !ok || got != reliable {
		t.Fatalf("first escalation = (%T, %v), want reliable provider", got, ok)
	}
	if router.got.Tier != ConversationTierReliable || router.got.EscalatedFrom != ConversationTierElevated {
		t.Fatalf("trust = %+v, want reliable from elevated", router.got)
	}
	if _, ok := a.EscalateConversationProvider(context.Background(), msg); ok {
		t.Fatal("second escalation must be rejected")
	}
	if router.calls != 2 {
		t.Fatalf("router calls = %d, want 2 ladder lookups with duplicate tier rejected", router.calls)
	}
	snap := a.LastRoutingSnapshot()
	if snap.ConversationTier != string(ConversationTierReliable) ||
		snap.ConversationEscalatedFrom != string(ConversationTierElevated) {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestEffectiveAIProviderAppliesConversationTrust(t *testing.T) {
	previous := GlobalChatRouting()
	t.Cleanup(func() { SetGlobalChatRouting(previous) })

	router := &trustRoutingStub{}
	SetGlobalChatRouting(router)
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{ID: "user", Name: "Camron"}, "Run the focused tests")

	if got := a.EffectiveAIProvider(context.Background(), msg); got == nil {
		t.Fatal("expected effective provider")
	}
	if router.got.Tier != ConversationTierElevated ||
		!conversationContainsString(router.got.Reasons, ConversationReasonExplicitToolAction) {
		t.Fatalf("router trust = %+v", router.got)
	}
	snap := a.LastRoutingSnapshot()
	if snap.ConversationTier != string(ConversationTierElevated) {
		t.Fatalf("snapshot tier = %q", snap.ConversationTier)
	}
}
