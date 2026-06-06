package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubPersistentStore struct {
	byChannel map[string][]*protocol.Message
}

func (s *stubPersistentStore) InsertMessage(msg *protocol.Message) error {
	if s.byChannel == nil {
		s.byChannel = map[string][]*protocol.Message{}
	}
	s.byChannel[msg.Channel] = append(s.byChannel[msg.Channel], msg)
	return nil
}

func (s *stubPersistentStore) ListChannelMessages(channel string, limit int, beforeID string) ([]*protocol.Message, error) {
	msgs := s.byChannel[channel]
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	out := make([]*protocol.Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (s *stubPersistentStore) ClearChannelMessages(channel string) error {
	delete(s.byChannel, channel)
	return nil
}

func TestClearChannelHistory_clearsPersistentStore(t *testing.T) {
	h := NewHub()
	name := "dm-persist-clear"
	h.CreateChannelWithType(name, "test", "", protocol.ChannelTypeDM, "system")
	stub := &stubPersistentStore{}
	h.SetPersistentMessageStore(stub)

	user := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	for i := 0; i < 3; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeChat, name, user, "hello")
		if err := h.SendMessage(msg); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(stub.byChannel[name]) >= 3 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(stub.byChannel[name]) < 3 {
		t.Fatalf("expected persisted messages, got %d", len(stub.byChannel[name]))
	}

	if err := h.ClearChannelHistory(name); err != nil {
		t.Fatal(err)
	}
	if len(stub.byChannel[name]) != 0 {
		t.Fatalf("persistent store should be empty after clear, got %d", len(stub.byChannel[name]))
	}

	older, err := h.GetMessagesPage(name, 50, "cursor-msg-id")
	if err != nil {
		t.Fatal(err)
	}
	if len(older) != 0 {
		t.Fatalf("GetMessagesPage should return no older rows after clear, got %d", len(older))
	}
}
