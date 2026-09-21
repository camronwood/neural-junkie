package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEnsureAwaitingUserTerminalRunCue(t *testing.T) {
	got := ensureAwaitingUserTerminalRunCue("Run this:\n```bash\nls\n```")
	if !strings.Contains(got, awaitingUserTerminalRunCue) {
		t.Fatalf("missing wait cue: %q", got)
	}
	again := ensureAwaitingUserTerminalRunCue(got)
	if strings.Count(again, awaitingUserTerminalRunCue) != 1 {
		t.Fatalf("cue should be idempotent, got %q", again)
	}
}

func TestApplySuggestedCommandsToResponse_addsCueAndSoftensCompleted(t *testing.T) {
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}, "run ls")
	responseMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "general", ag.Info, "")
	responseMsg.SetTaskStatus("completed")

	out := applySuggestedCommandsToResponse(ag, msg, responseMsg, "Please run:\n```bash\nls -la\n```\nTASK_STATUS: completed\n")
	if !strings.Contains(out, awaitingUserTerminalRunCue) {
		t.Fatalf("expected wait cue in %q", out)
	}
	if responseMsg.Metadata["suggested_commands"] == nil {
		t.Fatal("expected suggested_commands metadata")
	}
	if got := responseMsg.GetTaskStatus(); got != "in_progress" {
		t.Fatalf("task_status=%q, want in_progress while awaiting terminal", got)
	}
}

func TestShouldRespond_CommandOutputForSuggestingAgent(t *testing.T) {
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, shouldRespondTestHub{})
	ag.SetCollabClient(shouldRespondTestCollab{})

	out := protocol.NewMessage(
		protocol.MessageTypeCommandOutput,
		"general",
		protocol.AgentInfo{ID: "terminal", Name: "Terminal", Type: protocol.AgentTypeGeneral},
		"@BackendEngineer suggested a terminal command.",
	)
	out.Metadata = map[string]interface{}{
		"suggested_by_agent": "BackendEngineer",
	}
	if !ag.shouldRespond(out) {
		t.Fatal("suggesting agent should wake on command_output")
	}

	other := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"react"}, mockAI, shouldRespondTestHub{})
	other.SetCollabClient(shouldRespondTestCollab{})
	if other.shouldRespond(out) {
		t.Fatal("non-suggesting agent must not wake on command_output")
	}
}

func TestShouldRespond_CommandOutputViaReplyTo(t *testing.T) {
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, shouldRespondTestHub{})
	ag.SetCollabClient(shouldRespondTestCollab{})

	suggestion := protocol.NewMessage(protocol.MessageTypeAnswer, "general", ag.Info, "```bash\nls\n```")
	suggestion.Metadata = map[string]interface{}{
		"suggested_commands": []protocol.CommandSuggestion{{Command: "ls", AgentName: ag.Info.Name}},
	}
	ag.replaceChannelHistory("general", []*protocol.Message{suggestion})

	out := protocol.NewMessage(
		protocol.MessageTypeCommandOutput,
		"general",
		protocol.AgentInfo{ID: "terminal", Name: "Terminal", Type: protocol.AgentTypeGeneral},
		"result",
	)
	out.ReplyTo = suggestion.ID
	if !ag.shouldRespond(out) {
		t.Fatal("expected wake via reply_to suggestion message")
	}
}

func TestShouldRespond_CommandOutputWithoutTargetIgnored(t *testing.T) {
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, shouldRespondTestHub{})
	ag.SetCollabClient(shouldRespondTestCollab{})

	out := protocol.NewMessage(
		protocol.MessageTypeCommandOutput,
		"general",
		protocol.AgentInfo{ID: "terminal", Name: "Terminal", Type: protocol.AgentTypeGeneral},
		"random output",
	)
	if ag.shouldRespond(out) {
		t.Fatal("command_output without suggested_by_agent / reply_to must not wake agents")
	}
}

func TestApplySuggestedCommandsToResponse_skipsIllustrativeExample(t *testing.T) {
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}, "ui missing")
	responseMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "general", ag.Info, "")

	body := "Since you haven't shared files yet...\n\nExample fix:\n\n```bash\nnpm start\n```\n"
	out := applySuggestedCommandsToResponse(ag, msg, responseMsg, body)
	if strings.Contains(out, awaitingUserTerminalRunCue) {
		t.Fatalf("illustrative example must not append wait cue: %q", out)
	}
	if responseMsg.Metadata != nil && responseMsg.Metadata["suggested_commands"] != nil {
		t.Fatalf("expected no suggested_commands, got %#v", responseMsg.Metadata["suggested_commands"])
	}
}

func TestTerminalBashSuggestionPrompt_mentionsWait(t *testing.T) {
	p := TerminalBashSuggestionPrompt()
	for _, needle := range []string{"run_command", "NEVER say you cannot execute", "command_output", "```bash fenced blocks```", "workspace"} {
		if !strings.Contains(p, needle) {
			t.Fatalf("prompt missing %q: %s", needle, p)
		}
	}
}
