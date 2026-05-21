package confluence

import (
	"testing"
	"time"
)

func TestConfluenceIndexAddAndLookup(t *testing.T) {
	idx := NewConfluenceIndex("ENG", "Engineering")
	parent := &Page{ID: "p1", Title: "Parent", Content: "parent body", LastUpdated: time.Now()}
	child := &Page{ID: "p2", Title: "Child", Content: "child body", ParentID: "p1", Labels: []string{"api"}, LastUpdated: time.Now()}
	idx.AddPage(parent)
	idx.AddPage(child)

	if idx.PageCount != 2 {
		t.Fatalf("page count %d", idx.PageCount)
	}
	if got, ok := idx.GetPage("p2"); !ok || got.Title != "Child" {
		t.Fatal("get page")
	}
	byLabel := idx.GetPagesByLabel("api")
	if len(byLabel) != 1 || byLabel[0].ID != "p2" {
		t.Fatalf("by label: %+v", byLabel)
	}
	children := idx.GetChildPages("p1")
	if len(children) != 1 || children[0].ID != "p2" {
		t.Fatalf("children: %+v", children)
	}
	stats := idx.GetStats()
	if stats["page_count"] != 2 {
		t.Fatalf("stats: %+v", stats)
	}
}

func TestConfluenceIndexIsStale(t *testing.T) {
	idx := NewConfluenceIndex("X", "X")
	idx.LastIndexed = time.Now().Add(-48 * time.Hour)
	if !idx.IsStale(24 * time.Hour) {
		t.Fatal("expected stale index")
	}
	idx.LastIndexed = time.Now()
	if idx.IsStale(24 * time.Hour) {
		t.Fatal("expected fresh index")
	}
}
