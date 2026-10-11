package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type proactiveCaptureHub struct {
	shouldRespondTestHub
	mu   sync.Mutex
	msgs []*protocol.Message
}

func (h *proactiveCaptureHub) SendMessage(msg *protocol.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = append(h.msgs, msg)
	return nil
}

func (h *proactiveCaptureHub) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.msgs)
}

func TestProactiveSuggestionsDedupe(t *testing.T) {
	hub := &proactiveCaptureHub{}
	a := NewAssistantAgent("Assistant", ai.NewMockProvider(), hub)
	if a.storage == nil {
		t.Skip("assistant storage unavailable")
	}
	a.config.ProactiveAssistance = true

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{
		ID: "u1", Name: "User", Type: "human",
	}, "we have a meeting tomorrow and a todo for the launch")
	msg.ID = "m1"

	a.checkProactiveSuggestions(context.Background(), msg)
	first := hub.count()
	if first == 0 {
		t.Fatal("expected at least one proactive suggestion")
	}
	a.checkProactiveSuggestions(context.Background(), msg)
	if hub.count() != first {
		t.Fatalf("expected dedupe, got %d then %d", first, hub.count())
	}
}

func TestProactiveWiredViaInterceptor(t *testing.T) {
	hub := &proactiveCaptureHub{}
	a := NewAssistantAgent("Assistant", ai.NewMockProvider(), hub)
	if a.storage == nil {
		t.Skip("assistant storage unavailable")
	}
	a.config.ProactiveAssistance = true
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{
		ID: "u1", Name: "User", Type: "human",
	}, "remember this important note about the sync")
	msg.ID = "m-proactive-wire"

	// Call interceptor path directly (same wiring as production handleMessage).
	if a.messageInterceptor == nil {
		t.Fatal("expected message interceptor")
	}
	_ = a.messageInterceptor(context.Background(), msg)
	if hub.count() == 0 {
		t.Fatal("expected proactive suggestion via interceptor")
	}
}
