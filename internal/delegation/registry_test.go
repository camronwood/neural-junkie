package delegation

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestRelevanceScoreBiologyBeatsBackend(t *testing.T) {
	q := "analyze this dna sequence for mutations"
	backend := protocol.AgentInfo{Type: protocol.AgentTypeBackend, Expertise: []string{"APIs", "Go"}}
	bio := protocol.AgentInfo{Type: protocol.AgentTypeBiology, Expertise: []string{"molecular biology", "sequences"}}
	if RelevanceScore(bio, q) <= RelevanceScore(backend, q) {
		t.Fatalf("biology should score higher on bio question: bio=%d backend=%d",
			RelevanceScore(bio, q), RelevanceScore(backend, q))
	}
}

func TestResolveSkipsSelfAndLowerScores(t *testing.T) {
	from := protocol.AgentInfo{ID: "a1", Name: "GoExpert", Type: protocol.AgentTypeBackend, Expertise: []string{"Go", "APIs"}}
	candidates := []protocol.AgentInfo{
		from,
		{ID: "a2", Name: "BiologyExpert", Type: protocol.AgentTypeBiology, Expertise: []string{"biology", "protein"}},
	}
	q := "what does this protein sequence imply for expression"
	got := Resolve(from, q, candidates, ResolveOptions{MinScore: 2, MaxCandidates: 2})
	if len(got) != 1 || got[0].AgentName != "BiologyExpert" {
		t.Fatalf("expected BiologyExpert consult, got %+v", got)
	}
}

func TestLooksBioTools(t *testing.T) {
	if !looksBioTools("please fold_protein on this peptide") {
		t.Fatal("expected bio tools intent")
	}
}
