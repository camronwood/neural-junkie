package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestKeepLastPtrSlice(t *testing.T) {
	var nilSlice []*protocol.Message
	if got := keepLastPtrSlice(nilSlice, 5); got != nil {
		t.Fatalf("nil slice: got %v want nil", got)
	}
	msgs := make([]*protocol.Message, 10)
	for i := range msgs {
		msgs[i] = &protocol.Message{Content: string(rune('0' + i))}
	}
	trimmed := keepLastPtrSlice(msgs, 3)
	if len(trimmed) != 3 {
		t.Fatalf("len=%d want 3", len(trimmed))
	}
	if trimmed[0].Content != "7" {
		t.Fatalf("first kept should be oldest of last three")
	}
}

func TestAppendChannelMessageLockedTrimsAndKeepsOrder(t *testing.T) {
	h := NewHub()
	ch := "general"
	h.mu.Lock()
	for i := 0; i < MaxHubChannelHistory+50; i++ {
		m := protocol.NewMessage(protocol.MessageTypeChat, ch, protocol.AgentInfo{ID: "a", Name: "A", Type: protocol.AgentTypeGeneral}, string(rune(i)))
		h.appendChannelMessageLocked(ch, m)
	}
	got := len(h.messages[ch])
	h.mu.Unlock()
	if got != MaxHubChannelHistory {
		t.Fatalf("channel len=%d want %d", got, MaxHubChannelHistory)
	}
}
