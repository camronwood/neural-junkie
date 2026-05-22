package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type broadcastRecordingHub struct {
	shouldRespondTestHub
	mu         sync.Mutex
	broadcasts []*protocol.Message
}

func (h *broadcastRecordingHub) BroadcastDirect(_ string, msg *protocol.Message) {
	h.mu.Lock()
	h.broadcasts = append(h.broadcasts, msg)
	h.mu.Unlock()
}

func (h *broadcastRecordingHub) broadcastsOfType(t *testing.T, typ protocol.MessageType) []*protocol.Message {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []*protocol.Message
	for _, m := range h.broadcasts {
		if m.Type == typ {
			out = append(out, m)
		}
	}
	return out
}

func TestAbortChannelCancelsRegisteredContext(t *testing.T) {
	a := NewAgent(protocol.AgentTypeBackend, "AbortTest", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()
	id := RegisterGenCancelForTest(a, "general", genCancel)
	defer a.unregisterGenCancel("general", id)

	done := make(chan struct{})
	go func() {
		<-genCtx.Done()
		close(done)
	}()

	a.AbortChannel("general")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected generation context to be canceled")
	}
	if ActiveGenCountForTest(a, "general") != 0 {
		t.Fatalf("expected active gens cleared, got %d", ActiveGenCountForTest(a, "general"))
	}
}

func TestCollectStreamTokensReturnsCanceled(t *testing.T) {
	hubRec := &broadcastRecordingHub{}
	a := NewAgent(protocol.AgentTypeBackend, "StreamAbort", nil, ai.NewMockProvider(), hubRec)
	ctx, cancel := context.WithCancel(context.Background())

	tokenCh := make(chan ai.StreamToken, 2)
	tokenCh <- ai.StreamToken{Content: "partial "}
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{
		ID: "u1", Name: "User", Type: "human",
	}, "hi")
	msg.ID = "parent-msg-id"

	_, _, _, err := a.collectStreamTokens(ctx, msg, "stream-abc", tokenCh)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	ends := hubRec.broadcastsOfType(t, protocol.MessageTypeStreamEnd)
	if len(ends) != 1 {
		t.Fatalf("expected one stream_end, got %d", len(ends))
	}
	if ends[0].ID != "stream-abc" {
		t.Fatalf("stream_end id=%q", ends[0].ID)
	}
}

func TestCollectStreamTokensCanceledAppendsStoppedSuffix(t *testing.T) {
	hubRec := &broadcastRecordingHub{}
	a := NewAgent(protocol.AgentTypeBackend, "StreamStopSuffix", nil, ai.NewMockProvider(), hubRec)
	ctx, cancel := context.WithCancel(context.Background())

	tokenCh := make(chan ai.StreamToken, 4)
	tokenCh <- ai.StreamToken{Content: "hello"}
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{
		ID: "u1", Name: "User", Type: "human",
	}, "hi")
	msg.ID = "parent-2"

	_, _, _, err := a.collectStreamTokens(ctx, msg, "stream-2", tokenCh)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	deltas := hubRec.broadcastsOfType(t, protocol.MessageTypeStreamDelta)
	if len(deltas) < 1 || deltas[0].Content != "hello" {
		t.Fatalf("expected hello delta, got %d deltas", len(deltas))
	}
}
