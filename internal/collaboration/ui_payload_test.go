package collaboration

import (
	"encoding/json"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestToUIPayloadOmitsDiscussionMessages(t *testing.T) {
	c := &Collaboration{
		ID:    "c1",
		Title: "t",
		Phase: PhasePlanning,
		Discussion: &DiscussionSession{
			ID:              "d1",
			CollaborationID: "c1",
			Status:          DiscussionActive,
			Messages: []*protocol.Message{
				protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "a", Name: "A", Type: protocol.AgentTypeBackend}, "hi"),
			},
		},
	}
	ui := c.ToUIPayload()
	if ui == nil || ui.Discussion == nil {
		t.Fatal("expected discussion summary")
	}
	data, err := json.Marshal(ui)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) == "" {
		t.Fatal("empty payload")
	}
	// Full Collaboration marshal would include "hi" in nested messages; UI payload must not.
	if len(c.Discussion.Messages) > 0 {
		msgContent := c.Discussion.Messages[0].Content
		if msgContent != "" && json.Valid(data) {
			var m map[string]interface{}
			_ = json.Unmarshal(data, &m)
			disc, _ := m["discussion"].(map[string]interface{})
			if disc != nil {
				if _, has := disc["messages"]; has {
					t.Fatal("UI payload must not include discussion.messages")
				}
			}
		}
	}
}
