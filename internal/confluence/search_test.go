package confluence

import (
	"testing"
	"time"
)

func sampleIndex() *ConfluenceIndex {
	idx := NewConfluenceIndex("DOC", "Docs")
	now := time.Now()
	idx.AddPage(&Page{
		ID: "1", Title: "Authentication Guide", Content: "OAuth2 and JWT tokens for API access",
		Labels: []string{"security"}, LastUpdated: now,
	})
	idx.AddPage(&Page{
		ID: "2", Title: "Deployment", Content: "Kubernetes rollout steps",
		Labels: []string{"devops"}, LastUpdated: now.Add(-time.Hour),
	})
	idx.AddPage(&Page{
		ID: "3", Title: "API Reference", Content: "REST endpoints",
		ParentID: "1", Labels: []string{"api"}, LastUpdated: now,
		Comments: []PageComment{{ID: "c1", Content: "JWT refresh flow", Author: "camron", CreatedAt: now}},
	})
	return idx
}

func TestSearcherSearchAndTitle(t *testing.T) {
	s := NewSearcher(sampleIndex())
	results := s.Search("jwt oauth", 5)
	if len(results) == 0 || results[0].Page.ID != "1" {
		t.Fatalf("search jwt: %+v", results)
	}
	titleHits := s.SearchByTitle("api", 5)
	if len(titleHits) == 0 || titleHits[0].Page.ID != "3" {
		t.Fatalf("title search: %+v", titleHits)
	}
}

func TestSearcherLabelCommentsRelated(t *testing.T) {
	s := NewSearcher(sampleIndex())
	byLabel := s.SearchByLabel("api")
	if len(byLabel) != 1 || byLabel[0].Page.ID != "3" {
		t.Fatalf("label: %+v", byLabel)
	}
	comments := s.SearchInComments("refresh", 5)
	if len(comments) == 0 {
		t.Fatal("expected comment match")
	}
	related := s.FindRelatedPages("1", 5)
	if len(related) == 0 {
		t.Fatal("expected related pages for parent")
	}
}

func TestSearcherDateRangeAndRecent(t *testing.T) {
	s := NewSearcher(sampleIndex())
	now := time.Now()
	past := now.Add(-48 * time.Hour)
	future := now.Add(48 * time.Hour)
	inRange := s.SearchByDateRange(past, future)
	if len(inRange) < 2 {
		t.Fatalf("date range: %d", len(inRange))
	}
	recent := s.GetMostRecentPages(2)
	if len(recent) != 2 {
		t.Fatalf("recent: %d", len(recent))
	}
}

func TestSearcherEmptyQuery(t *testing.T) {
	s := NewSearcher(sampleIndex())
	if len(s.Search("", 10)) != 0 {
		t.Fatal("empty query should return no results")
	}
}
