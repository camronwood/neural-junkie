package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestWireRoomChannelAgentsJoinsGeneralAgents(t *testing.T) {
	h := NewHub()
	assistant := &protocol.AgentInfo{ID: "asst-1", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	if err := h.RegisterAgent(assistant); err != nil {
		t.Fatal(err)
	}
	if err := h.JoinChannel(assistant.ID, "general"); err != nil {
		t.Fatal(err)
	}

	room, err := h.CreateRoom("Host", DefaultRoomOptions())
	if err != nil {
		t.Fatal(err)
	}
	chName := RoomGeneralChannel(room.ID)
	h.CreateChannelWithType(chName, "Room chat", "", protocol.ChannelTypeRoom, "Host")

	h.WireRoomChannelAgents(chName)

	agents, err := h.GetChannelAgents(chName)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent in room channel, got %d", len(agents))
	}
	if agents[0].ID != assistant.ID {
		t.Fatalf("expected assistant wired, got %q", agents[0].ID)
	}
}

func TestRoomExpirySweeperEndsRooms(t *testing.T) {
	h := NewHub()
	room, err := h.CreateRoom("Host", RoomOptions{
		Name:       "Test",
		TTL:        10 * time.Millisecond,
		MaxMembers: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.JoinRoom(room.JoinCode, "Host"); err != nil {
		t.Fatal(err)
	}
	chName := RoomGeneralChannel(room.ID)
	h.CreateChannelWithType(chName, "Room chat", "", "room", "Host")
	h.SyncRoomChannelMembers(room.ID)

	time.Sleep(20 * time.Millisecond)
	h.expireRoomsOnce()

	if _, ok := h.GetRoom(room.ID); ok {
		t.Fatalf("expected room expired")
	}
	// Channel should be deleted on expiry sweep.
	if _, err := h.GetChannel(chName); err == nil {
		t.Fatalf("expected channel deleted on expiry")
	}
}

