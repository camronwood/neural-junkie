package export

import (
	"strings"
	"testing"
)

func TestParseJSONL(t *testing.T) {
	input := `{"instruction":"Hello","output":"World"}
{"instruction":"Q2","input":"ctx","output":"A2"}
`
	rows, err := ParseJSONL(strings.NewReader(input), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].SourceKind != "import" {
		t.Fatalf("expected import source kind, got %q", rows[0].SourceKind)
	}
}

func TestParseJSONLRejectsMissingOutput(t *testing.T) {
	_, err := ParseJSONL(strings.NewReader(`{"instruction":"only input"}`), 10)
	if err == nil {
		t.Fatal("expected error for missing output")
	}
}

func TestParsePastedRowsTabSeparated(t *testing.T) {
	rows, err := ParsePastedRows("Explain auth\tUse JWT middleware\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Instruction != "Explain auth" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}
