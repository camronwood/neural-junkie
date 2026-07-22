package protocol

import "testing"

func TestRoutingMetaConversationTrustRoundTrip(t *testing.T) {
	msg := &Message{}
	ApplyRoutingMeta(msg, RoutingMeta{
		Model:                     "reliable-model",
		ConversationTier:          "reliable",
		ConversationReasons:       []string{"quality_gate_failure"},
		ConversationEscalatedFrom: "elevated",
		Attempts: []RoutingAttempt{
			{ProviderID: "ollama-local", Model: "qwen2.5:3b", Tier: "standard", FailureReason: "quality_gate_failure"},
			{ProviderID: "ollama-local", Model: "qwen3.5:9b", Tier: "reliable", Reason: "local_escalation"},
		},
		FailureEvidence: []string{"quality_gate_failure"},
	})
	got := ExtractRoutingMeta(msg)
	if got.ConversationTier != "reliable" ||
		got.ConversationEscalatedFrom != "elevated" ||
		len(got.ConversationReasons) != 1 ||
		got.ConversationReasons[0] != "quality_gate_failure" ||
		len(got.Attempts) != 2 ||
		got.Attempts[0].FailureReason != "quality_gate_failure" ||
		len(got.FailureEvidence) != 1 {
		t.Fatalf("routing metadata = %+v", got)
	}
}
