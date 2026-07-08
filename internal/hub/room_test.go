package hub

import (
	"testing"
	"time"
)

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

