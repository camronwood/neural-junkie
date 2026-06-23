package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendLearningsForMessage_assistantPrompt(t *testing.T) {
	unlock := learning.LockTestGlobals()
	dir, err := os.MkdirTemp("", "nj-learning-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		unlock()
		_ = os.RemoveAll(dir)
	}()
	store, err := learning.NewStore(filepath.Join(dir, "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	learning.SetGlobalStore(store)
	learning.SetEnabledChecker(func() bool { return true })

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

func TestAppendLearningsForMessage_concurrentPerAgentCopy(t *testing.T) {
	unlock := learning.LockTestGlobals()
	defer unlock()
	learning.SetEnabledChecker(func() bool { return false })

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "collab-scenarios",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff")
	msg.Metadata = map[string]any{
		"workspace_id":        "ws-collab",
		"collaboration_id":    "abc",
		"collab_internal_event": true,
	}

	self := &protocol.AgentInfo{ID: "assistant-1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	const workers = 12
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			work, err := protocol.CloneMessage(msg)
			if err != nil {
				errCh <- err
				return
			}
			var sb strings.Builder
			AppendLearningsForMessage(&sb, work, self)
			errCh <- nil
		}()
	}
	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
}
