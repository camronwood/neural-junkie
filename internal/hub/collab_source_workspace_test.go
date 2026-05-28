package hub

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollaborateSkipsCollabSandboxAsSourceWorkspace(t *testing.T) {
	h := NewHub()
	assetsRoot := t.TempDir()
	h.SetCollaborationAssetsRootResolver(func() string { return assetsRoot })

	sandbox := filepath.Join(assetsRoot, "71bc548f-da3e-4485-834a-b6fc7ddbfa15")
	if err := os.MkdirAll(sandbox, 0755); err != nil {
		t.Fatalf("mkdir sandbox: %v", err)
	}

	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatalf("command handler: %v", err)
	}
	h.commandHandler = ch

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human", Name: "tester", Type: protocol.AgentTypeGeneral},
		"/collaborate @Gemini @Assistant investigate schemas",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "Collab: investigate",
			"workspace_path": sandbox,
			"file_tree":      "",
			"open_files":     []interface{}{},
		},
	}

	if out, err := ch.handleCollaborate(context.Background(), msg, strings.Fields(msg.Content)); err != nil || out != nil {
		t.Fatalf("handleCollaborate out=%v err=%v", out, err)
	}

	active := h.GetCollaborationManager().ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active collaboration, got %d", len(active))
	}
	if got := active[0].SourceRepoPath; got != "" {
		t.Fatalf("source workspace = %q, want empty for collab sandbox", got)
	}
}

func TestCollaborateSkipsProjectCollabDeliverablePathAsSourceWorkspace(t *testing.T) {
	h := NewHub()
	collabDir := filepath.Join(t.TempDir(), "Phoenix", "collabs", "902f2cf4-0626-4726-835a-4f1b715c23f6")
	if err := os.MkdirAll(collabDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path, warn := h.resolveCollaborateSourceRepoPath(&protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": collabDir,
			},
		},
	}, collaborateFlagParse{})
	if path != "" {
		t.Fatalf("path = %q, want empty for project collabs/<id> folder", path)
	}
	if warn == "" || !strings.Contains(warn, "deliverables") {
		t.Fatalf("expected deliverables-folder warning, got %q", warn)
	}
}

func TestCollaborateSkipsCollabReviewsPathAsSourceWorkspace(t *testing.T) {
	h := NewHub()
	assetsRoot := t.TempDir()
	h.SetCollaborationAssetsRootResolver(func() string { return assetsRoot })

	reviewsDir := filepath.Join(assetsRoot, "reviews", "3ec2d77e-1270-4a47-8a6d-06ff6ae0abb7")
	if err := os.MkdirAll(reviewsDir, 0755); err != nil {
		t.Fatalf("mkdir reviews: %v", err)
	}

	path, warn := h.resolveCollaborateSourceRepoPath(&protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": reviewsDir,
			},
		},
	}, collaborateFlagParse{})
	if path != "" {
		t.Fatalf("path = %q, want empty", path)
	}
	if warn == "" {
		t.Fatal("expected warning for reviews path")
	}
}
