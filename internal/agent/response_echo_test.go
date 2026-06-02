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
