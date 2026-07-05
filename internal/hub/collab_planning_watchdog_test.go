package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTickPlanningDiscussionWatchdog_HandoffForSilentParticipant(t *testing.T) {
	h := newTestHub(t)
	chName := "watchdog-planning"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{
		MaxRounds:        1,
		MaxTotalMessages: 4,
	})
	if err != nil {
		t.Fatal(err)
	}

	discMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		chName,
		protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant},
		"first plan line",
	)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Add(collabPlanningHandoffRedispatchAfter + time.Second)
	h.tickPlanningDiscussionWatchdog(collab, now)

	msgs, _ := h.GetMessages(chName, 50)
	handoffs := 0
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if m.Type != protocol.MessageTypeCollabDiscussion || !m.IsFromSystem() {
			continue
		}
		if !m.IsMentioned("a2") {
			continue
		}
		if body := m.Content; body != "" && (strings.Contains(body, "Collaboration turn handoff") || strings.Contains(body, "You're up first")) {
			handoffs++
		}
	}
	if handoffs == 0 {
		t.Fatal("expected watchdog to send planning turn handoff for silent participant")
	}
}
