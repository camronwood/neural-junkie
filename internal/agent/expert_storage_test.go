package agent

import (
	"path/filepath"
	"testing"
)

func TestExpertAgentStorage_ListWithMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "expert-agents.json")
	storage := &ExpertAgentStorage{path: path}

	record := ExpertAgentRecord{
		AgentID:      "agent-1",
		Name:         "Swift-Coding-iOSExpert",
		ExpertSlug:   "ios",
		ProviderName: "ollama",
		Model:        "gemma3:12b",
		CreatedBy:    "camronwood",
		DMChannel:    "dm-camronwood-swift-coding-iosexpert",
		Created:      "2026-07-01T19:57:03Z",
	}
	if err := storage.Save(record); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rows, err := storage.ListWithMetadata()
	if err != nil {
		t.Fatalf("ListWithMetadata: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d; want 1", len(rows))
	}
	row := rows[0]
	if row["type"] != "expert" {
		t.Fatalf("type = %v; want expert", row["type"])
	}
	if row["name"] != record.Name {
		t.Fatalf("name = %v", row["name"])
	}
	meta, ok := row["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata type = %T", row["metadata"])
	}
	if meta["expert_slug"] != "ios" || meta["dm_channel"] != record.DMChannel {
		t.Fatalf("metadata = %#v", meta)
	}

	deleted, err := storage.DeleteByName(record.Name)
	if err != nil {
		t.Fatalf("DeleteByName: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to succeed")
	}
	rows, err = storage.ListWithMetadata()
	if err != nil {
		t.Fatalf("ListWithMetadata after delete: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("len(rows) after delete = %d; want 0", len(rows))
	}
}
