package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestLooksLikeEchoOfPriorUserTurn(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "What?")
	msg.ID = "cur"
	hist := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"},
			"I want to add theme supoort to this project"),
	}
	if !looksLikeEchoOfPriorUserTurn(msg, "I want to add theme supoort to this project.", hist) {
		t.Fatal("expected echo of prior user line")
	}
	if !looksLikeEchoOfPriorUserTurn(msg, "The user wants to add theme support to their project.", nil) {
		t.Fatal("expected session-summary style echo")
	}
	if looksLikeEchoOfPriorUserTurn(msg, "I don't have your workspace files in context yet — enable workspace sharing.", nil) {
		t.Fatal("expected valid answer")
	}
}

func TestLooksLikeReAskAfterAffirmation(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "approved")
	msg.ID = "cur"
	hist := []*protocol.Message{
		{
			ID:   "fc1",
			Type: protocol.MessageTypeFileChange,
			From: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
			Content: "edit src/App.tsx",
		},
	}
	reask := "Great! Let's move forward with implementing the settings button. Could you provide more details on the icon design?"
	if !looksLikeReAskAfterAffirmation(msg, reask, hist) {
		t.Fatal("expected re-ask after approval")
	}
	action := "I'll update src/components/SettingsButton.tsx with the gear icon and wire the theme toggle now."
	if looksLikeReAskAfterAffirmation(msg, action, hist) {
		t.Fatal("expected action reply to pass")
	}
	shareReask := "Great! Let's pick up where we left off. Can you share the current implementation of the settings modal?"
	if !looksLikeReAskAfterAffirmation(msg, shareReask, hist) {
		t.Fatal("expected share-the-files re-ask after approval")
	}
}

func TestLooksLikeAsksUserToPasteWorkspaceFiles(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "fix blank screen")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{"workspace_path": "/proj", "workspace_name": "proj"},
	}
	paste := "Could you please share the content of vite.config.ts?"
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, paste) {
		t.Fatal("expected paste request with workspace shared")
	}
	ok := "Grounding: I loaded 3 file(s). The issue is in src/main.tsx — missing root mount."
	if looksLikeAsksUserToPasteWorkspaceFiles(msg, ok) {
		t.Fatal("expected grounded answer to pass")
	}
	msg.Metadata = nil
	if looksLikeAsksUserToPasteWorkspaceFiles(msg, paste) {
		t.Fatal("expected no workspace context to skip")
	}
}

func TestLooksLikePrematureFileApplyClaim(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "save to docs/test.md")
	msg.ID = "cur"
	claim := "The article has been created and saved to docs/test.md."
	if !looksLikePrematureFileApplyClaim(msg, claim, nil) {
		t.Fatal("expected premature apply claim")
	}
	proposal := "I submitted a file change proposal for your approval."
	if looksLikePrematureFileApplyClaim(msg, proposal, nil) {
		t.Fatal("expected proposal wording to pass")
	}
	hist := []*protocol.Message{
		{
			ID:      "sys1",
			Type:    protocol.MessageTypeSystemInfo,
			Content: "Applied change `fc1` to `docs/test.md`.",
		},
	}
	if looksLikePrematureFileApplyClaim(msg, claim, hist) {
		t.Fatal("expected prior Applied change to suppress flag")
	}
}
