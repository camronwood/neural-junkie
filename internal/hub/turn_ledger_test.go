package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/turnledger"
)

func TestChannelMaintainsTurnLedger_collab(t *testing.T) {
	if !channelMaintainsTurnLedger(protocol.ChannelTypeCollaboration, "collab-abc") {
		t.Fatal("collaboration channels should maintain turn ledger")
	}
	if channelMaintainsTurnLedger(protocol.ChannelTypePublic, "chat-scenarios") {
		t.Fatal("harness channels must skip turn ledger")
	}
	if !channelMaintainsSessionSummary(protocol.ChannelTypeCollaboration, "collab-abc") {
		t.Fatal("collaboration channels should maintain session summary")
	}
}

func TestNoteTurnLedgerAndRead(t *testing.T) {
	dir := t.TempDir()
	turnledger.SetDirForTest(dir)
	t.Cleanup(func() { turnledger.SetDirForTest("") })

	h := NewHub()
	h.channels["general"] = &protocol.Channel{Name: "general", Type: protocol.ChannelTypePublic}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: "human"},
		"Please review `ThemeSettings`")
	h.noteTurnLedger(msg)

	deadline := time.Now().Add(2 * time.Second)
	var rows []interface{}
	for time.Now().Before(deadline) {
		got := h.GetChannelTurnLedger("general", 10)
		if len(got) > 0 {
			if got[0].Speaker != "Camron" {
				t.Fatalf("speaker=%q", got[0].Speaker)
			}
			if got[0].SpeakerType != "human" {
				t.Fatalf("type=%q", got[0].SpeakerType)
			}
			found := false
			for _, e := range got[0].Entities {
				if e == "ThemeSettings" {
					found = true
				}
			}
			if !found {
				t.Fatalf("entities=%v", got[0].Entities)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
		_ = rows
	}
	t.Fatal("timed out waiting for async turn ledger append")
}
