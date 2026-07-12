package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSubscribeTier_StreamDeltaReachesUIOnly(t *testing.T) {
	h := NewHub()
	const ch = "tier-stream"
	_ = h.CreateChannel(ch, "tier stream", "test")

	agentSub, err := h.Subscribe(ch)
	if err != nil {
		t.Fatalf("Subscribe agent: %v", err)
	}
	defer h.Unsubscribe(ch, agentSub)

	uiSub, err := h.SubscribeUI(ch)
	if err != nil {
		t.Fatalf("SubscribeUI: %v", err)
	}
	defer h.UnsubscribeUI(ch, uiSub)

	delta := protocol.NewMessage(
		protocol.MessageTypeStreamDelta,
		ch,
		protocol.AgentInfo{ID: "a1", Name: "Agent", Type: protocol.AgentTypeGeneral},
		"tok",
	)
	h.BroadcastDirect(ch, delta)

	select {
	case got := <-uiSub:
		if got.Type != protocol.MessageTypeStreamDelta || got.Content != "tok" {
			t.Fatalf("UI got %+v, want stream_delta tok", got)
		}
	case <-time.After(time.Second):
		t.Fatal("UI subscriber did not receive stream_delta")
	}

	select {
	case m := <-agentSub:
		t.Fatalf("agent subscriber should not receive stream_delta, got %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSubscribeTier_DurableMessageReachesBoth(t *testing.T) {
	h := NewHub()
	const ch = "tier-chat"
	_ = h.CreateChannel(ch, "tier chat", "test")

	agentSub, err := h.Subscribe(ch)
	if err != nil {
		t.Fatalf("Subscribe agent: %v", err)
	}
	defer h.Unsubscribe(ch, agentSub)

	uiSub, err := h.SubscribeUI(ch)
	if err != nil {
		t.Fatalf("SubscribeUI: %v", err)
	}
	defer h.UnsubscribeUI(ch, uiSub)

	chat := protocol.NewMessage(
		protocol.MessageTypeChat,
		ch,
		protocol.AgentInfo{ID: "u1", Name: "User", Type: protocol.AgentTypeGeneral},
		"hello",
	)
	if err := h.SendMessage(chat); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	for i, sub := range []struct {
		name string
		ch   chan *protocol.Message
	}{
		{"agent", agentSub},
		{"ui", uiSub},
	} {
		select {
		case got := <-sub.ch:
			if got.Type != protocol.MessageTypeChat || got.Content != "hello" {
				t.Fatalf("%s subscriber got %+v, want chat hello", sub.name, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s subscriber did not receive chat (attempt %d)", sub.name, i)
		}
	}
}

func TestSubscribeTier_ControlPlaneAgentStatusReachesAgent(t *testing.T) {
	h := NewHub()
	const ch = "tier-control"
	_ = h.CreateChannel(ch, "tier control", "test")

	agentSub, err := h.Subscribe(ch)
	if err != nil {
		t.Fatalf("Subscribe agent: %v", err)
	}
	defer h.Unsubscribe(ch, agentSub)

	uiSub, err := h.SubscribeUI(ch)
	if err != nil {
		t.Fatalf("SubscribeUI: %v", err)
	}
	defer h.UnsubscribeUI(ch, uiSub)

	control := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		ch,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"",
	)
	control.Metadata = map[string]interface{}{
		protocol.MetadataChannelHold: true,
	}
	h.BroadcastDirect(ch, control)

	for _, sub := range []struct {
		name string
		ch   chan *protocol.Message
	}{
		{"agent", agentSub},
		{"ui", uiSub},
	} {
		select {
		case got := <-sub.ch:
			if got.Type != protocol.MessageTypeAgentStatus {
				t.Fatalf("%s got type %q, want agent_status", sub.name, got.Type)
			}
			if v, ok := got.Metadata[protocol.MetadataChannelHold].(bool); !ok || !v {
				t.Fatalf("%s got metadata %+v, want channel_hold=true", sub.name, got.Metadata)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s subscriber did not receive control-plane agent_status", sub.name)
		}
	}
}

func TestSubscribeTier_UIOnlyAgentStatusSkippedByAgent(t *testing.T) {
	h := NewHub()
	const ch = "tier-ui-status"
	_ = h.CreateChannel(ch, "tier ui status", "test")

	agentSub, err := h.Subscribe(ch)
	if err != nil {
		t.Fatalf("Subscribe agent: %v", err)
	}
	defer h.Unsubscribe(ch, agentSub)

	uiSub, err := h.SubscribeUI(ch)
	if err != nil {
		t.Fatalf("SubscribeUI: %v", err)
	}
	defer h.UnsubscribeUI(ch, uiSub)

	uiStatus := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		ch,
		protocol.AgentInfo{ID: "a1", Name: "Agent", Type: protocol.AgentTypeGeneral},
		"thinking",
	)
	uiStatus.Metadata = map[string]interface{}{
		protocol.MetadataTelemetryKind: "turn_trace",
	}
	h.BroadcastDirect(ch, uiStatus)

	select {
	case got := <-uiSub:
		if got.Type != protocol.MessageTypeAgentStatus {
			t.Fatalf("UI got type %q, want agent_status", got.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("UI subscriber did not receive UI-only agent_status")
	}

	select {
	case m := <-agentSub:
		t.Fatalf("agent subscriber should not receive UI-only agent_status, got %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
}
