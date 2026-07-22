package sqlite

import (
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestOpenReopenAfterInserts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "messages.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeChat, "c", protocol.AgentInfo{ID: "t", Name: "t", Type: "human"}, "hi")
		if err := store.InsertMessage(msg); err != nil {
			t.Fatal(err)
		}
	}
	reply := protocol.NewMessage(protocol.MessageTypeSystemInfo, "c", protocol.AgentInfo{ID: "a", Name: "Agent", Type: "frontend"}, "error")
	reply.ReplyTo = "parent-id"
	if err := store.InsertMessage(reply); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen after inserts: %v", err)
	}
	defer reopened.Close()
	msgs, err := reopened.ListChannelMessages("c", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 6 {
		t.Fatalf("got %d messages, want 6", len(msgs))
	}
	var found bool
	for _, m := range msgs {
		if m.ReplyTo == "parent-id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected reply_to to round-trip through sqlite")
	}
}

func TestBackfillDoesNotLockWhileCursorOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "messages.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	msg := protocol.NewMessage(protocol.MessageTypeChat, "c", protocol.AgentInfo{ID: "t", Name: "t", Type: "human"}, "hi")
	if err := store.InsertMessage(msg); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM messages_fts`); err != nil {
		t.Fatal(err)
	}
	if err := store.ensureFTSBackfill(); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	n, err := store.ftsRowCount()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("fts rows=%d want 1", n)
	}
}

func TestOpenDefaultMessagesDB(t *testing.T) {
	// Opens the real default path only as a smoke check. Skip when the DB is
	// empty (fresh CI runners / new installs) — FTS backfill is covered by
	// TestBackfillDoesNotLockWhileCursorOpen with an isolated temp DB.
	path, err := PathDefault()
	if err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer store.Close()
	var msgCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&msgCount); err != nil {
		t.Fatal(err)
	}
	if msgCount == 0 {
		t.Skipf("no messages in %s", path)
	}
	n, err := store.ftsRowCount()
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatalf("expected FTS backfill on %s (%d messages), got 0 FTS rows", path, msgCount)
	}
}
