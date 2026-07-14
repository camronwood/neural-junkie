package collaboration

import "testing"

func TestFocusScopedDeliverableInventoryHit(t *testing.T) {
	allowed := []string{"README.md", "core/sample/main.go"}
	body := "# findings\n- README is a fixture\n- main.go has HelloWorld\n"
	if hit := FocusScopedDeliverableInventoryHit(body, allowed); hit != "" {
		t.Fatalf("unexpected hit %q", hit)
	}
	bad := body + "- Also notes src/App.tsx and React components elsewhere.\n"
	if hit := FocusScopedDeliverableInventoryHit(bad, allowed); hit == "" {
		t.Fatal("expected inventory hit for App.tsx or src/App.tsx")
	}
	server := "- The core/server/main.go package is separate.\n"
	if hit := FocusScopedDeliverableInventoryHit(server, allowed); hit == "" {
		t.Fatal("expected core/server/main.go hit")
	}
}

func TestDeliverablePolicyWrappers(t *testing.T) {
	p := NewDeliverablePolicy(
		CollaborationTask{Title: "Write findings.md", Description: "summarize citing sources"},
		"summarize findings",
		[]string{"README.md"},
	)
	if !p.RequiresFile() {
		t.Fatal("expected RequiresFile")
	}
	if !p.MarkdownOnly() {
		t.Fatal("expected MarkdownOnly")
	}
	if !p.ResearchFindings() {
		t.Fatal("expected ResearchFindings")
	}
	if p.Kind() != DeliverableKindMarkdown {
		t.Fatalf("kind=%q want markdown", p.Kind())
	}
	if p.RequiresImplementationSession() {
		t.Fatal("markdown deliverable must not require implementation session")
	}

	code := NewDeliverablePolicy(
		CollaborationTask{Title: "Implement handler", Description: "Create cmd/server/foo.go"},
		"",
		nil,
	)
	if code.Kind() != DeliverableKindFile {
		t.Fatalf("kind=%q want file", code.Kind())
	}
	if !code.RequiresImplementationSession() {
		t.Fatal("coding file deliverable requires implementation session")
	}

	none := NewDeliverablePolicy(
		CollaborationTask{Title: "Discuss approach", Description: "Share thoughts in chat"},
		"",
		nil,
	)
	if none.Kind() != DeliverableKindNone {
		t.Fatalf("kind=%q want none", none.Kind())
	}
}
