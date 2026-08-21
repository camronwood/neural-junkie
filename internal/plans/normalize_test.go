package plans

import (
	"strings"
	"testing"
)

const gemmaPseudoPlan = `Based on my research of docs/index.html, here's a structured YAML plan to add a HelloWorld function: yaml plan: add-helloworld-function-to-index-html target_file: docs/index.html actions: - description: Insert a self-contained JavaScript console.log function before </body> location: at the end of <body>, before </html> implementation: content: | <script> window.HelloWorld = function() { console.log("Hello, World!"); }; </script> - rationale: - Self-contained, non-breaking addition`

const qwenPseudoPlan = `Plan for Adding HelloWorld Function to main.go yaml plan: project_name: dickory-docs status: No Go files exist steps: - description: Investigate if main.go should be created action: check_project_structure - description: Create new main.go with HelloWorld function action: create_file`

func TestNormalizeMarkdown_gemmaPseudoYAML(t *testing.T) {
	out, ok := NormalizeMarkdown(gemmaPseudoPlan)
	if !ok {
		t.Fatal("expected normalization")
	}
	doc, err := Parse(out)
	if err != nil {
		t.Fatalf("parse normalized: %v\n%s", err, out)
	}
	if doc.Name == "" {
		t.Fatalf("name empty: %+v", doc)
	}
	if len(doc.Todos) == 0 {
		t.Fatalf("expected todos, got %+v", doc)
	}
	if !strings.Contains(doc.Overview, "docs/index.html") {
		t.Fatalf("overview=%q", doc.Overview)
	}
	if !strings.Contains(doc.Body, "Out of scope") {
		t.Fatalf("body missing out of scope: %q", doc.Body)
	}
}

func TestNormalizeMarkdown_qwenPseudoYAML(t *testing.T) {
	out, ok := NormalizeMarkdown(qwenPseudoPlan)
	if !ok {
		t.Fatal("expected normalization")
	}
	doc, err := Parse(out)
	if err != nil {
		t.Fatalf("parse normalized: %v\n%s", err, out)
	}
	if len(doc.Todos) < 2 {
		t.Fatalf("todos=%d want >=2", len(doc.Todos))
	}
}

func TestNormalizeMarkdown_alreadyValid(t *testing.T) {
	out, ok := NormalizeMarkdown(timShaped)
	if ok {
		t.Fatal("valid plan should not rewrite")
	}
	if out != timShaped {
		t.Fatal("content changed")
	}
}

func TestPrepareMarkdown_roundTripStore(t *testing.T) {
	s := NewStore(t.TempDir())
	prepared := PrepareMarkdown(gemmaPseudoPlan)
	rec, err := s.SaveFromMarkdown(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || rec.ID == "" {
		t.Fatal("expected saved plan")
	}
}
