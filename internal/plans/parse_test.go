package plans

import (
	"strings"
	"testing"
)

const timShaped = `---
name: TIM org migrate wizard
overview: Add an interactive bash wizard that prints and optionally runs migrate_org_to_tim.py.
todos:
  - id: add-wizard-sh
    content: Add Phoenix/scripts/tenant/tim_org_migrate_wizard.sh
    status: pending
  - id: docs-links
    content: Link wizard from operator docs
    status: pending
isProject: false
---

# TIM org migration wizard

## Out of scope

- Changing orchestration order
`

func TestParseTIMShaped(t *testing.T) {
	doc, err := Parse(timShaped)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Name != "TIM org migrate wizard" {
		t.Fatalf("name=%q", doc.Name)
	}
	if len(doc.Todos) != 2 {
		t.Fatalf("todos=%d", len(doc.Todos))
	}
	if !strings.Contains(doc.Body, "Out of scope") {
		t.Fatalf("body=%q", doc.Body)
	}
}

func TestParseFencedMarkdown(t *testing.T) {
	fenced := "```markdown\n" + timShaped + "\n```"
	doc, err := Parse(fenced)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Todos[0].ID != "add-wizard-sh" {
		t.Fatalf("todo id=%q", doc.Todos[0].ID)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse("just a chat outline\n1. do a thing"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Parse("---\nname: x\n---\nno todos"); err == nil {
		t.Fatal("expected missing todos")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	s := NewStore(t.TempDir())
	rec, err := s.SaveFromMarkdown(timShaped)
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || rec.ID == "" {
		t.Fatal("expected saved record")
	}
	got, err := s.Get(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != rec.Name || len(got.Todos) != 2 {
		t.Fatalf("got %+v", got)
	}
	listed, err := s.List()
	if err != nil || len(listed) != 1 {
		t.Fatalf("list=%v err=%v", listed, err)
	}
	updated := strings.Replace(timShaped, "status: pending", "status: completed", 1)
	if _, err := s.Put(rec.ID, updated); err != nil {
		t.Fatal(err)
	}
	got, err = s.Get(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Todos[0].Status != "completed" {
		t.Fatalf("status=%q", got.Todos[0].Status)
	}
}

func TestSaveFromMarkdownIgnoresNonPlan(t *testing.T) {
	s := NewStore(t.TempDir())
	rec, err := s.SaveFromMarkdown("Outline:\n1. Add HelloWorld")
	if err != nil {
		t.Fatal(err)
	}
	if rec != nil {
		t.Fatalf("unexpected record %+v", rec)
	}
}
