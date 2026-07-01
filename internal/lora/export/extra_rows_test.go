package export

import "testing"

func TestAppendExtraRowsPreservesSourceKinds(t *testing.T) {
	merged := AppendExtraRows(
		[]Row{{Instruction: "learn", Output: "prefs", SourceKind: "learning"}},
		[]Row{{Instruction: "custom", Output: "answer", SourceKind: "import"}},
	)
	if len(merged) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(merged))
	}
	if merged[0].SourceKind != "learning" || merged[1].SourceKind != "import" {
		t.Fatalf("unexpected kinds: %#v", merged)
	}
	rows := mergeExtraRows(nil, merged)
	if len(rows) != 2 {
		t.Fatalf("expected 2 preview rows, got %d", len(rows))
	}
	if rows[0].SourceKind != "learning" || rows[1].SourceKind != "import" {
		t.Fatalf("unexpected preview kinds: %#v", rows)
	}
}

func TestCollectRowsWithMixedExtraAndRowIDs(t *testing.T) {
	chat := []PreviewRow{{
		Row: Row{
			RowID:       "chat1",
			Instruction: "hi",
			Output:      "hello",
			SourceKind:  "channel",
			Included:    true,
		},
	}}
	extras := []Row{{
		Instruction: "imported",
		Output:      "row",
		SourceKind:  "import",
	}}
	merged := mergeExtraRows(chat, extras)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged rows, got %d", len(merged))
	}
	filtered := applyRowFilters(merged, Request{RowIDs: []string{merged[0].RowID}})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 curated row, got %d", len(filtered))
	}
}
