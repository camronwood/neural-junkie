package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestWorkspaceDocumentReviewGuidanceFromStamp(t *testing.T) {
	msg := &protocol.Message{
		Content: "please reivew the documents in the workspace",
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/tmp/ciso",
				"file_tree":      "internal/\n  GILEAD.md\ngilead-security/\n  plan.md\n",
				"open_files": []interface{}{
					map[string]interface{}{
						"path":    "internal/GILEAD.md",
						"content": "# Gilead remediation\nReal guidance lives here.",
					},
				},
			},
			"context_scope": "outline",
		},
	}
	decision := intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionQuestion,
		RequestedAction: intent.ActionInspect,
		Action:          intent.ActionInspect,
		Retrieval:       []intent.RetrievalTarget{intent.RetrievalCodebase},
		Mutation:        intent.MutationNone,
		Confidence:      0.9,
		Source:          intent.SourceLocalModel,
		ReasonCodes:     []string{"project_overview"},
		ContextPlan: intent.ContextPlan{
			Tier:       intent.ContextTierOutline,
			Subject:    intent.ContextSubjectWorkspaceDocuments,
			ReviewMode: intent.ReviewModeWorkspace,
		},
	}
	if err := protocol.StampTurnDecision(msg, decision); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	appendWorkspaceReviewGuidance(&b, msg)
	out := b.String()
	if !strings.Contains(out, "WORKSPACE DOCUMENT REVIEW") {
		t.Fatalf("expected workspace document review guidance, got:\n%s", out)
	}
	if !strings.Contains(out, "Do NOT invent paths") {
		t.Fatalf("expected anti-hallucination guidance, got:\n%s", out)
	}
}
