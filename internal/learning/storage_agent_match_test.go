package learning

import (
	"path/filepath"
	"testing"
)

func TestListFiltered_agentMatchByTypeAndName(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Add(Entry{
		Scope:     ScopeAgent,
		UserID:    "camron",
		AgentID:   "old-assistant-uuid",
		AgentType: "assistant",
		AgentName: "Assistant",
		Content:   "My name is Camron.",
		Category:  CategoryFact,
	})
	if err != nil {
		t.Fatal(err)
	}

	rows := store.ListFiltered(Filter{
		AgentID:       "new-assistant-uuid",
		AgentType:     "assistant",
		AgentName:     "Assistant",
		UserID:        "camronwood",
		IncludeLegacy: true,
	})
	if len(rows) != 1 {
		t.Fatalf("expected 1 learning by type+name+user resolve, got %d", len(rows))
	}
}

func TestEntryMatchesAgent(t *testing.T) {
	e := Entry{Scope: ScopeAgent, AgentID: "old-id", AgentType: "assistant", AgentName: "Assistant"}
	if !entryMatchesAgent(e, "new-id", "assistant", "Assistant") {
		t.Fatal("expected match by type+name")
	}
	if entryMatchesAgent(e, "other-id", "backend", "Assistant") {
		t.Fatal("expected no match for wrong type")
	}
}
