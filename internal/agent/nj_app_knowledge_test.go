package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMessageAsksAboutNJApp(t *testing.T) {
	cases := []struct {
		content string
		want    bool
	}{
		{"what keyboard shortcuts does neural junkie have?", true},
		{"how do I open settings?", true},
		{"how do I toggle the terminal?", true},
		{"what is the command palette shortcut?", true},
		{"tell me about the dev pack and IDE layout", true},
		{"remind me in 5 minutes to review the PR", false},
		{"what model are you running?", false},
		{"how do I fix this golang error?", false},
	}
	for _, tc := range cases {
		got := messageAsksAboutNJApp(tc.content)
		if got != tc.want {
			t.Fatalf("messageAsksAboutNJApp(%q) = %v, want %v", tc.content, got, tc.want)
		}
	}
}

func TestBuildAssistantPrompt_includesNJAppKnowledge(t *testing.T) {
	a := &AssistantAgent{
		Agent: &Agent{
			Info: protocol.AgentInfo{
				Name:       "Assistant",
				Type:       protocol.AgentTypeAssistant,
				AIProvider: "ollama",
				AIModel:    "qwen2.5:7b",
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general",
		protocol.AgentInfo{ID: "u", Name: "Camron"}, "what keyboard shortcuts are in neural junkie?")
	prompt := a.buildAssistantPrompt(msg)
	if !strings.Contains(prompt, "NEURAL JUNKIE APP KNOWLEDGE (full reference)") {
		t.Fatal("expected full NJ app knowledge for shortcut question")
	}
	if !strings.Contains(prompt, "mod+shift+p command palette") {
		t.Fatal("expected command palette shortcut in full knowledge")
	}
}

func TestBuildAssistantPrompt_alwaysIncludesBriefNJKnowledge(t *testing.T) {
	a := &AssistantAgent{
		Agent: &Agent{
			Info: protocol.AgentInfo{
				Name:       "Assistant",
				Type:       protocol.AgentTypeAssistant,
				AIProvider: "ollama",
				AIModel:    "qwen2.5:7b",
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "u", Name: "Camron"}, "remind me tomorrow about standup")
	prompt := a.buildAssistantPrompt(msg)
	if !strings.Contains(prompt, "NEURAL JUNKIE APP (brief)") {
		t.Fatal("expected brief NJ knowledge in every assistant prompt")
	}
	if strings.Contains(prompt, "NEURAL JUNKIE APP KNOWLEDGE (full reference)") {
		t.Fatal("did not expect full NJ knowledge for unrelated reminder")
	}
}
