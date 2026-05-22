package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestChannelHoldSetClearAndHumanSendClears(t *testing.T) {
	h := NewHub()
	h.CreateChannel("hold-test", "", "")
	if h.IsChannelHeld("hold-test") {
		t.Fatal("expected not held initially")
	}
	h.SetChannelHold("hold-test", true, "user1")
	if !h.IsChannelHeld("hold-test") {
		t.Fatal("expected held after set")
	}
	h.SetChannelHold("hold-test", false, "")
	if h.IsChannelHeld("hold-test") {
		t.Fatal("expected cleared")
	}

	h.SetChannelHold("hold-test", true, "user1")
	msg := protocol.NewMessage(protocol.MessageTypeChat, "hold-test", protocol.AgentInfo{
		ID: "u1", Name: "Camron", Type: "human",
	}, "hello")
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if h.IsChannelHeld("hold-test") {
		t.Fatal("human send should clear channel hold")
	}
}

func TestInterjectChannelNotFound(t *testing.T) {
	h := NewHub()
	err := h.InterjectChannel("missing", "")
	if err == nil {
		t.Fatal("expected error for missing channel")
	}
}
