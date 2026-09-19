package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEmitEditApplyStreamChunksAndEnd(t *testing.T) {
	h := NewHub()
	chName := "edit-apply-stream-test"
	_ = h.CreateChannel(chName, "test", "public")
	ui, err := h.SubscribeUI(chName)
	if err != nil {
		t.Fatalf("SubscribeUI: %v", err)
	}
	defer h.UnsubscribeUI(chName, ui)

	content := strings.Repeat("abcde", 80) // 400 bytes → multiple 256-byte chunks
	from := protocol.AgentInfo{ID: "a1", Name: "Agent", Type: "agent"}
	go h.EmitEditApplyStream(chName, from, "chg-stream-1", "/ws/src/a.ts", content)

	deadline := time.After(2 * time.Second)
	var deltas []*protocol.Message
	var end *protocol.Message
	joined := ""
	for end == nil {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for stream_end; got %d deltas", len(deltas))
		case m := <-ui:
			if m == nil {
				continue
			}
			switch m.Type {
			case protocol.MessageTypeStreamDelta:
				meta, ok := protocol.ParseEditApplyStreamMeta(m.Metadata)
				if !ok {
					t.Fatalf("delta missing edit_apply meta: %+v", m.Metadata)
				}
				if meta.ChangeID != "chg-stream-1" || meta.Path != "/ws/src/a.ts" {
					t.Fatalf("unexpected meta: %+v", meta)
				}
				if meta.Offset != len(joined) {
					t.Fatalf("offset %d want %d", meta.Offset, len(joined))
				}
				joined += m.Content
				deltas = append(deltas, m)
			case protocol.MessageTypeStreamEnd:
				meta, ok := protocol.ParseEditApplyStreamMeta(m.Metadata)
				if !ok || !meta.Done {
					t.Fatalf("stream_end meta: ok=%v %+v", ok, meta)
				}
				end = m
			}
		}
	}

	if len(deltas) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(deltas))
	}
	if joined != content {
		t.Fatalf("rejoined content length %d want %d", len(joined), len(content))
	}
	if end.ID != deltas[0].ID {
		t.Fatalf("stream id mismatch end=%s delta=%s", end.ID, deltas[0].ID)
	}
}

func TestEmitEditApplyStreamSkipsEmpty(t *testing.T) {
	h := NewHub()
	chName := "edit-apply-empty"
	_ = h.CreateChannel(chName, "test", "public")
	ui, err := h.SubscribeUI(chName)
	if err != nil {
		t.Fatalf("SubscribeUI: %v", err)
	}
	defer h.UnsubscribeUI(chName, ui)

	h.EmitEditApplyStream(chName, protocol.AgentInfo{Name: "A"}, "c1", "a.ts", "")
	select {
	case m := <-ui:
		t.Fatalf("unexpected message: %+v", m)
	case <-time.After(50 * time.Millisecond):
	}
}
