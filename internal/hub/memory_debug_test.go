package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHubMemoryReport(t *testing.T) {
	h := NewHub()
	h.mu.Lock()
	h.messages["general"] = append(h.messages["general"], &protocol.Message{
		ID:      "1",
		Type:    protocol.MessageTypeChat,
		Channel: "general",
		From:    protocol.AgentInfo{ID: "a", Name: "A", Type: protocol.AgentTypeGeneral},
		Content: "hello",
	})
	h.mu.Unlock()

	rep := h.HubMemoryReport()
	if rep["hub_total_msgs"].(int) != 1 {
		t.Fatalf("hub_total_msgs=%v", rep["hub_total_msgs"])
	}
	if rep["hub_content_bytes"].(int64) != 5 {
		t.Fatalf("hub_content_bytes=%v", rep["hub_content_bytes"])
	}
}
