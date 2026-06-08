package hub

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCreateDMChannel_EagerSubscribeAnswersFirstMessage(t *testing.T) {
	h := NewHub()
	cursor := agent.NewAgent(protocol.AgentTypeCLI, "Cursor", nil, ai.NewMockProvider(), h)
	cursor.Info.ID = "cursor-test-id"
	h.commandHandler.RegisterRuntimeAgent(cursor)
	if err := h.RegisterAgent(&cursor.Info); err != nil {
		t.Fatal(err)
	}
	if err := cursor.Start(context.Background(), "general"); err != nil {
		t.Fatal(err)
	}

	dm, err := h.CreateDMChannel("camron", cursor.Info.ID)
	if err != nil {
		t.Fatal(err)
	}

	userMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		dm.Name,
		protocol.AgentInfo{ID: "human-1", Name: "camron", Type: "human"},
		"you here?",
	)
	if err := h.SendMessage(userMsg); err != nil {
		t.Fatal(err)
	}

	waitForAgentChatReply(t, h, dm.Name, cursor.Info.ID)
}

func TestCreateDMChannel_ReplayMissedFirstMessage(t *testing.T) {
	h := NewHub()
	cursor := agent.NewAgent(protocol.AgentTypeCLI, "Cursor", nil, ai.NewMockProvider(), h)
	cursor.Info.ID = "cursor-test-id"
	h.commandHandler.RegisterRuntimeAgent(cursor)
	if err := h.RegisterAgent(&cursor.Info); err != nil {
		t.Fatal(err)
	}

	dmName := "dm-camron-cursor"
	ch := h.CreateChannelWithType(
		dmName,
		"Direct message with Cursor",
		"",
		protocol.ChannelTypeDM,
		"camron",
	)
	if err := h.JoinChannel(cursor.Info.ID, ch.Name); err != nil {
		t.Fatal(err)
	}

	userMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		dmName,
		protocol.AgentInfo{ID: "human-1", Name: "camron", Type: "human"},
		"you here?",
	)
	if err := h.SendMessage(userMsg); err != nil {
		t.Fatal(err)
	}

	if err := cursor.Start(context.Background(), "general"); err != nil {
		t.Fatal(err)
	}
	h.ensureAgentSubscribed(cursor.Info.ID, dmName)

	waitForAgentChatReply(t, h, dmName, cursor.Info.ID)
}

func waitForAgentChatReply(t *testing.T, h *Hub, channel, agentID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		msgs, err := h.GetMessages(channel, 20)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range msgs {
			if m.From.ID == agentID && m.Type == protocol.MessageTypeChat {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected agent %s chat reply in %s", agentID, channel)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
