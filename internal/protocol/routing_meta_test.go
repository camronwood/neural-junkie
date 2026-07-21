package protocol

import "testing"

func TestRoutingMetaConversationTrustRoundTrip(t *testing.T) {
	msg := &Message{}
	ApplyRoutingMeta(msg, RoutingMeta{
		Model:                     "reliable-model",
		ConversationTier:          "reliable",
		ConversationReasons:       []string{"quality_gate_failure"},
		ConversationEscalatedFrom: "elevated",
	})
	got := ExtractRoutingMeta(msg)
	if got.ConversationTier != "reliable" ||
		got.ConversationEscalatedFrom != "elevated" ||
		len(got.ConversationReasons) != 1 ||
		got.ConversationReasons[0] != "quality_gate_failure" {
		t.Fatalf("routing metadata = %+v", got)
	}
}
