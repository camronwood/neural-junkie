package export

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubMsgs struct {
	msgs []*protocol.Message
}

func (s stubMsgs) GetMessages(string, int) ([]*protocol.Message, error) {
	return s.msgs, nil
}

func (s stubMsgs) GetThreadMessages(string, int) ([]*protocol.Message, error) {
	return s.msgs, nil
}

func TestExportChannelPairs(t *testing.T) {
	user := protocol.AgentInfo{Name: "Camron", Type: "human"}
	agent := protocol.AgentInfo{Name: "BackendEngineer", Type: protocol.AgentTypeBackend}
	msgs := []*protocol.Message{
		{Type: protocol.MessageTypeChat, From: user, Content: "How do I add auth?"},
		{Type: protocol.MessageTypeAnswer, From: agent, Content: "Use middleware and JWT."},
	}
	n, err := PreviewRowCount(Request{Source: SourceChannel, SourceID: "general", MaxRows: 100}, stubMsgs{msgs: msgs}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rows = %d", n)
	}
}

func TestExportCollaborationTasks(t *testing.T) {
	collab := &collaboration.Collaboration{
		Tasks: []collaboration.CollaborationTask{
			{Title: "Threat model", Description: "Review auth flow", Output: "Use OAuth2 with PKCE."},
		},
	}
	// pad to meet minimum
	for i := 0; i < MinRows; i++ {
		collab.Tasks = append(collab.Tasks, collaboration.CollaborationTask{
			Title: "T", Description: "prompt", Output: "answer",
		})
	}
	dir := t.TempDir()
	path := dir + "/dataset.jsonl"
	n, err := Export(Request{Source: SourceCollaboration, SourceID: "abc", MaxRows: 100}, stubMsgs{}, collab, path)
	if err != nil {
		t.Fatal(err)
	}
	if n < MinRows {
		t.Fatalf("rows = %d", n)
	}
}
