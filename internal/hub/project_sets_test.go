package hub

import (
	"path/filepath"
	"testing"
)

func TestProjectSetManager_CreateListDelete(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	wm, err := NewWorkspaceManager()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ws, err := wm.AddWorkspace("primary", filepath.Join(dir, "primary"), AddWorkspaceOptions{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	ws2, err := wm.AddWorkspace("linked", filepath.Join(dir, "linked"), AddWorkspaceOptions{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	pm, err := NewProjectSetManager()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := pm.CreateProjectSet("TestSet", ws.ID, []string{ws2.ID}, wm)
	if err != nil {
		t.Fatal(err)
	}
	if ps.Name != "TestSet" {
		t.Fatalf("name=%q", ps.Name)
	}
	if len(pm.ListProjectSets()) != 1 {
		t.Fatalf("list len=%d", len(pm.ListProjectSets()))
	}
	if err := pm.DeleteProjectSet(ps.ID); err != nil {
		t.Fatal(err)
	}
	if len(pm.ListProjectSets()) != 0 {
		t.Fatal("expected empty list after delete")
	}
}
