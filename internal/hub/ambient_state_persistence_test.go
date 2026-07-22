package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type ambientCaptureStore struct {
	inserted *protocol.Message
}

func (s *ambientCaptureStore) InsertMessage(msg *protocol.Message) error {
	s.inserted = msg
	return nil
}
func (s *ambientCaptureStore) ListChannelMessages(string, int, string) ([]*protocol.Message, error) {
	return nil, nil
}
func (s *ambientCaptureStore) SearchMessages(MessageSearchOptions) ([]*protocol.Message, error) {
	return nil, nil
}
func (s *ambientCaptureStore) ClearChannelMessages(string) error { return nil }

func TestAmbientStateIsStrippedFromAllPersistence(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test", protocol.AgentInfo{}, "debug this")
	msg.Metadata[agent.MetadataAmbientState] = map[string]interface{}{
		"terminal": map[string]interface{}{"failed_tail": "failure"},
	}

	sessionCopy := cloneMessageForSessionPersist(msg)
	if _, exists := sessionCopy.Metadata[agent.MetadataAmbientState]; exists {
		t.Fatal("session snapshot retained ambient state")
	}

	store := &ambientCaptureStore{}
	h := NewHub()
	h.CreateChannelWithType("dm-test", "test", "", protocol.ChannelTypeDM, "system")
	h.mu.Lock()
	h.messages["dm-test"] = []*protocol.Message{msg}
	h.mu.Unlock()
	h.SetPersistentMessageStore(store)
	h.persistMessage(msg)
	if store.inserted == nil {
		t.Fatal("expected persistent insert")
	}
	if _, exists := store.inserted.Metadata[agent.MetadataAmbientState]; exists {
		t.Fatal("durable message store retained ambient state")
	}
	if _, exists := msg.Metadata[agent.MetadataAmbientState]; !exists {
		t.Fatal("persistence stripping mutated the live message")
	}
}
