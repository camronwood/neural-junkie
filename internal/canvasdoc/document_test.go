package canvasdoc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompileMarkdownFixtures(t *testing.T) {
	src := strings.TrimSpace(`
# Trip plan

Welcome to the page.

## Places

- Tokyo
- Kyoto

1. Pack
2. Fly

| Name | Role |
| --- | --- |
| Ada | Architect |
| Grace | Engineer |

` + "```mermaid\nflowchart LR\n  A --> B\n```" + `

![Skyline](/api/artifacts/a1/assets/embed.png)

Leftover prose stays markdown.
`)
	doc := CompileMarkdown(src)
	if doc.SchemaVersion != 1 {
		t.Fatalf("schema=%d", doc.SchemaVersion)
	}
	types := make([]string, 0, len(doc.Blocks))
	for _, b := range doc.Blocks {
		types = append(types, b.Type)
	}
	want := []string{
		TypeHeading, TypeMarkdown, TypeHeading, TypeList, TypeList,
		TypeTable, TypeMermaid, TypeImage, TypeMarkdown,
	}
	if strings.Join(types, ",") != strings.Join(want, ",") {
		t.Fatalf("types=%v want=%v", types, want)
	}
	if doc.Blocks[0].Text != "Trip plan" || doc.Blocks[0].Level != 1 {
		t.Fatalf("heading=%+v", doc.Blocks[0])
	}
	if !containsAll(doc.Blocks[3].Items, "Tokyo", "Kyoto") {
		t.Fatalf("list=%+v", doc.Blocks[3].Items)
	}
	if !doc.Blocks[4].Ordered || doc.Blocks[4].Items[0] != "Pack" {
		t.Fatalf("ordered=%+v", doc.Blocks[4])
	}
	table := doc.Blocks[5]
	if len(table.Columns) != 2 || table.Columns[0].Key != "name" || table.Rows[0]["name"] != "Ada" {
		t.Fatalf("table=%+v", table)
	}
	if !strings.Contains(doc.Blocks[6].Source, "flowchart LR") {
		t.Fatalf("mermaid=%q", doc.Blocks[6].Source)
	}
	if doc.Blocks[7].Src != "/api/artifacts/a1/assets/embed.png" || doc.Blocks[7].Alt != "Skyline" {
		t.Fatalf("image=%+v", doc.Blocks[7])
	}
}

func TestUnwrapLegacyMarkdownString(t *testing.T) {
	payload, err := json.Marshal("# Canvas\n\n- one\n")
	if err != nil {
		t.Fatal(err)
	}
	doc := Unwrap(payload)
	if len(doc.Blocks) < 2 || doc.Blocks[0].Type != TypeHeading || doc.Blocks[1].Type != TypeList {
		t.Fatalf("unwrap=%+v", doc.Blocks)
	}
}

func TestUnwrapContentObject(t *testing.T) {
	payload, err := json.Marshal(map[string]string{"content": "## Notes\n\nhello"})
	if err != nil {
		t.Fatal(err)
	}
	doc := Unwrap(payload)
	if FirstHeading(doc) != "Notes" {
		t.Fatalf("heading=%q doc=%+v", FirstHeading(doc), doc)
	}
}

func TestParseDocumentJSON(t *testing.T) {
	raw := `{"schema_version":1,"blocks":[{"type":"heading","level":1,"text":"Hi"}]}`
	doc, ok := Parse(raw)
	if !ok || FirstHeading(doc) != "Hi" {
		t.Fatalf("parse ok=%v doc=%+v", ok, doc)
	}
	fromModel := FromModelOutput("```json\n" + raw + "\n```")
	if FirstHeading(fromModel) != "Hi" {
		t.Fatalf("fenced json=%+v", fromModel)
	}
}

func TestFromModelOutputFallsBackToMarkdown(t *testing.T) {
	doc := FromModelOutput("# Title\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n")
	var sawTable bool
	for _, b := range doc.Blocks {
		if b.Type == TypeTable && b.Rows[0]["a"] == "1" {
			sawTable = true
		}
	}
	if !sawTable {
		t.Fatalf("expected compiled table: %+v", doc.Blocks)
	}
}

func TestToMarkdownRoundTripCore(t *testing.T) {
	src := "# Trip plan\n\n- Tokyo\n\n| Name | Role |\n| --- | --- |\n| Ada | Architect |\n"
	md := ToMarkdown(CompileMarkdown(src))
	if !strings.Contains(md, "# Trip plan") || !strings.Contains(md, "- Tokyo") || !strings.Contains(md, "Ada") {
		t.Fatalf("markdown=%q", md)
	}
}

func TestIsPageRenderer(t *testing.T) {
	if !IsPageRenderer(RendererID, MediaType) || !IsPageRenderer("nj.markdown", "text/markdown") {
		t.Fatal("expected page renderers")
	}
	if IsPageRenderer("nj.mermaid", "text/vnd.mermaid") {
		t.Fatal("mermaid is not a page")
	}
}

func containsAll(items []string, want ...string) bool {
	set := map[string]bool{}
	for _, item := range items {
		set[item] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}
