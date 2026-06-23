package sqlite

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSearchMessagesFTS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "messages.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for i := 0; i < 100; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u1", Name: "User"}, "hello world baseline")
		msg.ID = "msg-" + string(rune('a'+i%26)) + "-" + time.Now().Format("150405")
		msg.Timestamp = time.Now().Add(time.Duration(i) * time.Millisecond)
		if err := store.InsertMessage(msg); err != nil {
			t.Fatal(err)
		}
	}
	needle := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u2", Name: "Camron"}, "unique needle phrase for search")
	needle.ID = "needle-1"
	if err := store.InsertMessage(needle); err != nil {
		t.Fatal(err)
	}

	results, err := store.SearchWithOptions(SearchOptions{Channel: "general", Query: "needle", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "needle-1" {
		t.Fatalf("results = %+v", results)
	}
	_ = os.Getenv("HOME")
}
