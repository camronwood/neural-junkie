package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsEditorDocumentReview(t *testing.T) {
	if !userRequestsEditorDocumentReview("I have a new document open, can you reivew?") {
		t.Fatal("expected review intent with typo")
	}
	if userRequestsEditorDocumentReview("remind me in 5 minutes") {
		t.Fatal("reminder should not be editor review")
	}
}

func TestAppendAssistantWorkspaceReviewGuidance_FocusWithFiles(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-assistant", protocol.AgentInfo{Name: "Camron"}, "please review the open doc")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeFocus,
		"workspace_context": map[string]interface{}{
			"workspace_name": "sandbox",
			"open_files": []interface{}{
				map[string]interface{}{
					"path": "/proj/rfc.md", "language": "markdown", "content": "# RFC\n", "is_active": true,
				},
			},
		},
	}
	var b strings.Builder
	appendAssistantWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "DOCUMENT / CODE REVIEW") {
		t.Fatalf("expected review guidance, got %q", out)
	}
	if !strings.Contains(out, "Do NOT say you cannot access") {
		t.Fatal("expected no-access denial guidance")
	}
}

func TestAppendAssistantWorkspaceReviewGuidance_HintWithoutFiles(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "Camron"}, "review what's in my editor")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeHint,
		"workspace_context": map[string]interface{}{
			"workspace_name": "sandbox",
			"workspace_path": "/proj",
		},
	}
	var b strings.Builder
	appendAssistantWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "EDITOR CONTEXT (limited)") {
		t.Fatalf("expected limited hint guidance, got %q", out)
	}
}
