package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendLearningsForMessage_assistantPrompt(t *testing.T) {
	dir := t.TempDir()
	store, err := learning.NewStore(dir + "/learnings.json")
	if err != nil {
		t.Fatal(err)
	}
	learning.SetGlobalStore(store)
	learning.SetEnabledChecker(func() bool { return true })
	t.Cleanup(func() {
		learning.SetGlobalStore(nil)
		learning.SetEnabledChecker(nil)
	})

	_, err = store.Add(learning.Entry{
		Scope:    learning.ScopeAgent,
		UserID:   "camron",
		AgentID:  "assistant-1",
		Content:  "My name is Camron.",
		Category: learning.CategoryFact,
	})
	if err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general",
		protocol.AgentInfo{ID: "human-1", Name: "camronwood", Type: "human"},
		"What is my name?")
	msg.Metadata = map[string]any{MetadataHubSessionUsername: "Camron"}

	var sb strings.Builder
	pr := AppendLearningsForMessage(&sb, msg, &protocol.AgentInfo{
		ID:   "assistant-1",
		Name: "Assistant",
		Type: protocol.AgentTypeAssistant,
	})
	if pr.Count == 0 {
		t.Fatal("expected learnings injected for assistant")
	}
	out := sb.String()
	if !strings.Contains(out, "My name is Camron.") {
		t.Fatalf("expected learning content in prompt: %q", out)
	}
}

func TestBuildLearningPromptContext_sessionUsername(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "u1", Name: "camronwood", Type: "human"},
		"hi")
	msg.Metadata = map[string]any{MetadataHubSessionUsername: "Camron"}
	pctx := buildLearningPromptContext(msg)
	if pctx.UserID != "camron" {
		t.Fatalf("UserID = %q, want camron", pctx.UserID)
	}
}
