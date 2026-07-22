package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestGetChannelMessagesMerged_tailLimit(t *testing.T) {
	h := NewHub()
	ch := "dm-merged"
	_ = h.CreateChannelWithType(ch, "test", "", protocol.ChannelTypeDM, "system")
	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeChat, ch, user, "msg")
		_ = h.SendMessage(msg)
	}
	merged, err := h.GetChannelMessagesMerged(ch, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 3 {
		t.Fatalf("len=%d want 3", len(merged))
	}
}

func TestGetChannelMessagesMerged_boundedPersistentPage(t *testing.T) {
	h := NewHub()
	ch := "dm-merged-persist"
	_ = h.CreateChannelWithType(ch, "test", "", protocol.ChannelTypeDM, "system")
	stub := &stubPersistentStore{byChannel: map[string][]*protocol.Message{}}
	h.SetPersistentMessageStore(stub)
	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	for i := 0; i < 20; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeChat, ch, user, "msg")
		_ = stub.InsertMessage(msg)
	}
	merged, err := h.GetChannelMessagesMerged(ch, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 5 {
		t.Fatalf("len=%d want 5", len(merged))
	}
	if n := stub.channelLen(ch); n != 20 {
		t.Fatalf("store should remain intact, got %d", n)
	}
}
