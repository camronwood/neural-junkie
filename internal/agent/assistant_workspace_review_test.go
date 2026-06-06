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

func TestAppendWorkspaceReviewGuidance_FocusWithFiles(t *testing.T) {
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
	appendWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "DOCUMENT / CODE REVIEW") {
		t.Fatalf("expected review guidance, got %q", out)
	}
	if !strings.Contains(out, "Do NOT say you cannot access") {
		t.Fatal("expected no-access denial guidance")
	}
}

func TestAppendWorkspaceReviewGuidance_ProjectReviewOutline(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "Camron"},
		"Can you review the code in the workspace?")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeOutline,
		"workspace_context": map[string]interface{}{
			"workspace_name": "dickory-docs",
			"workspace_path": "/proj/dickory-docs",
			"file_tree":      "src/\n  App.tsx\n",
		},
	}
	var b strings.Builder
	appendWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "PROJECT CODE REVIEW") {
		t.Fatalf("expected project review guidance, got %q", out)
	}
	if !strings.Contains(out, "Do NOT ask for a single file path") {
		t.Fatal("expected no file-path nag guidance")
	}
}

func TestAppendWorkspaceReviewGuidance_HintWithoutFiles(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "Camron"}, "review what's in my editor")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeHint,
		"workspace_context": map[string]interface{}{
			"workspace_name": "sandbox",
			"workspace_path": "/proj",
		},
	}
	var b strings.Builder
	appendWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "EDITOR CONTEXT (limited)") {
		t.Fatalf("expected limited hint guidance, got %q", out)
	}
}

func TestAppendWorkspaceReviewGuidance_BiologyCanSeeFileQuestion(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-biologyexpert", protocol.AgentInfo{Name: "Camron"}, "can you see the file I have open in the editor?")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeFocus,
		"workspace_context": map[string]interface{}{
			"workspace_name": "scan",
			"open_files": []interface{}{
				map[string]interface{}{
					"path": "imageMetadata.json", "language": "text", "content": "", "is_active": true,
				},
			},
			"scan_summary": map[string]interface{}{
				"summary_dir": "",
				"wells_count": float64(96),
			},
		},
	}
	var b strings.Builder
	appendWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "naming exactly what is visible") {
		t.Fatalf("expected precise visibility guidance, got %q", out)
	}
	if !strings.Contains(out, "image pixels were not attached") {
		t.Fatalf("expected image pixel caveat, got %q", out)
	}
}
