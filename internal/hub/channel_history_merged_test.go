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
