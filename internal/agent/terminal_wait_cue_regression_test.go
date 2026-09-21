package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestApplySuggestedCommands_skipsIncidentalProseWithoutBashFence(t *testing.T) {
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}, "UI does not come up")
	responseMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "general", ag.Info, "")

	prose := `Hello Camron! Since you haven't shared the specific code files yet, I can't debug the exact error directly.
When you run ` + "`npm start`" + ` (or your specific startup command), look at your terminal output.
Paste those details here, and we can get that UI up and running!`

	out := applySuggestedCommandsToResponse(ag, msg, responseMsg, prose)
	if strings.Contains(out, awaitingUserTerminalRunCue) {
		t.Fatalf("wait cue must not attach for incidental inline commands:\n%s", out)
	}
	if responseMsg.Metadata["suggested_commands"] != nil {
		t.Fatalf("expected no suggested_commands, got %#v", responseMsg.Metadata["suggested_commands"])
	}
}

func TestLooksLikeAsksUserToPaste_flagsHaventSharedFiles(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}, "debug my desktop app")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{"workspace_path": "/tmp/dickory-docs", "file_tree": "src/\npackage.json"},
	}
	resp := "Since you haven't shared the specific code files yet, I can't debug the exact error directly. Please share main.js."
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, resp) {
		t.Fatal("expected paste/workspace-denial detection when workspace is shared")
	}
}
