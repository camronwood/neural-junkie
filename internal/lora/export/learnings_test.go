package export

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/learning"
)

func TestExportLearningsRows(t *testing.T) {
	rows := ExportLearningsRows([]learning.Entry{
		{AgentName: "Alpha", Category: learning.CategoryPreference, Content: "Use tabs", Scope: learning.ScopeAgent},
		{Category: learning.CategoryFact, Content: "Dark mode", Scope: learning.ScopeGlobal},
	})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Instruction == "" || rows[0].Output != "Use tabs" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}

func TestMergeLearningsRows(t *testing.T) {
	chat := []Row{{Instruction: "chat", Output: "reply"}}
	learn := []Row{{Instruction: "learn", Output: "pref"}}
	merged := MergeLearningsRows(chat, learn)
	if len(merged) != 2 || merged[0].Instruction != "learn" {
		t.Fatalf("expected learnings first: %+v", merged)
	}
}
