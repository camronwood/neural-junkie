package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMergeMessagePagesPrefersMemoryAndKeepsArchive(t *testing.T) {
	old := &protocol.Message{
		ID:        "a",
		Channel:   "dm",
		Content:   "from archive",
		Type:      protocol.MessageTypeChat,
		Timestamp: time.Unix(1, 0),
	}
	newerMem := &protocol.Message{
		ID:        "a",
		Channel:   "dm",
		Content:   "from memory",
		Type:      protocol.MessageTypeChat,
		Timestamp: time.Unix(1, 0),
	}
	extraArchive := &protocol.Message{
		ID:        "b",
		Channel:   "dm",
		Content:   "older archive only",
		Type:      protocol.MessageTypeChat,
		Timestamp: time.Unix(2, 0),
	}
	out := mergeMessagePages([]*protocol.Message{newerMem}, []*protocol.Message{old, extraArchive}, 50)
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	byID := map[string]string{}
	for _, m := range out {
		byID[m.ID] = m.Content
	}
	if byID["a"] != "from memory" {
		t.Fatalf("memory should win for id a, got %q", byID["a"])
	}
	if byID["b"] != "older archive only" {
		t.Fatalf("archive-only row missing: %#v", byID)
	}
}
