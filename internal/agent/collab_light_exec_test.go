package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabTaskPrefersLightExecution_Markdown(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "body")
	msg.Metadata = map[string]interface{}{
		"task_title":       "Write findings",
		"task_description": "Write collabs/x/findings.md with three bullets",
	}
	if !collabTaskPrefersLightExecution(msg) {
		t.Fatal("expected markdown deliverable to prefer light execution")
	}
}

func TestCollabTaskPrefersLightExecution_DeliverableKindMetadata(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "Create foo.go somehow")
	msg.Metadata = map[string]interface{}{
		"deliverable_kind": "markdown",
		"task_title":       "Write doc",
		"task_description": "Write collabs/x/notes.md",
	}
	if !collabTaskPrefersLightExecution(msg) {
		t.Fatal("deliverable_kind=markdown must prefer light exec without body scraping")
	}
	msg.Metadata["deliverable_kind"] = "file"
	if collabTaskPrefersLightExecution(msg) {
		t.Fatal("deliverable_kind=file must not prefer light exec")
	}
}

func TestCollabTaskPrefersLightExecution_Coding(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "c", protocol.AgentInfo{}, "body")
	msg.Metadata = map[string]interface{}{
		"task_title":       "Implement handler",
		"task_description": "Create cmd/server/foo.go with HTTP handler",
	}
	// Has write+go file — still a file deliverable; markdown-only check should be false
	// unless only .md. Mixed: TaskLooksLikeMarkdownDeliverable false if no .md
	if collabTaskPrefersLightExecution(msg) {
		t.Fatal("coding .go task should not prefer light markdown path")
	}
}
