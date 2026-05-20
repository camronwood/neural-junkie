package collaboration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListAndLoadRunbookTemplate(t *testing.T) {
	dir := t.TempDir()
	payload := `{
  "name": "demo",
  "title": "Demo",
  "description": "from template",
  "execution_policy": {"max_concurrent_tasks": 2},
  "tasks": [{"id": "t1", "title": "Step", "kind": "action", "action": {"type": "web_search", "config": {"query": "x"}}}]
}`
	if err := os.WriteFile(filepath.Join(dir, "demo.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ListRunbookTemplates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "demo" {
		t.Fatalf("list = %#v", list)
	}

	tpl, err := LoadRunbookTemplate(dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Description != "from template" || tpl.ExecutionPolicy.MaxConcurrentTasks != 2 {
		t.Fatalf("loaded = %#v", tpl)
	}
	if len(tpl.Tasks) != 1 || tpl.Tasks[0].Kind != TaskKindAction {
		t.Fatalf("tasks = %#v", tpl.Tasks)
	}
}
