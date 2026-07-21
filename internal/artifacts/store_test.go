package artifacts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDefaultRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, err := DefaultRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".neural-junkie", "artifacts")
	if root != want {
		t.Fatalf("DefaultRoot() = %q, want %q", root, want)
	}
}

func TestCreateGetUpdateAndRevisions(t *testing.T) {
	store := newTestStore(t)
	input := sampleArtifact("canvas-1")
	created, err := store.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Revision != 1 || created.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("unexpected created version: %#v", created)
	}
	if created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("unexpected timestamps: %#v", created)
	}

	// Returned values and input buffers must not alias store state.
	input.Payload[0] = 'x'
	created.Payload[0] = 'x'
	got, err := store.Get("canvas-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Payload) != `{"nodes":[1]}` {
		t.Fatalf("payload mutated through caller: %s", got.Payload)
	}

	got.Title = "updated"
	got.Payload = json.RawMessage(`{"nodes":[1,2]}`)
	updated, err := store.Update(*got, 1)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.Title != "updated" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	if !updated.CreatedAt.Equal(got.CreatedAt) || updated.UpdatedAt.Before(updated.CreatedAt) {
		t.Fatal("update did not preserve valid timestamps")
	}

	first, err := store.GetRevision("canvas-1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Artifact.Title != "Example" || first.Artifact.Revision != 1 {
		t.Fatalf("revision one changed: %#v", first)
	}
	revisions, err := store.ListRevisions("canvas-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 || revisions[0].Revision != 1 || revisions[1].Revision != 2 {
		t.Fatalf("unexpected revisions: %#v", revisions)
	}
}

func TestOptimisticConflictAndConcurrentUpdates(t *testing.T) {
	store := newTestStore(t)
	created, err := store.Create(sampleArtifact("conflict"))
	if err != nil {
		t.Fatal(err)
	}
	created.Title = "first"
	if _, err := store.Update(*created, 99); !errors.Is(err, ErrConflict) {
		t.Fatalf("Update error = %v, want ErrConflict", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, title := range []string{"a", "b"} {
		wg.Add(1)
		go func(title string) {
			defer wg.Done()
			copy := *created
			copy.Title = title
			_, err := store.Update(copy, 1)
			results <- err
		}(title)
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected update error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestListAndFilters(t *testing.T) {
	store := newTestStore(t)
	first := sampleArtifact("b")
	first.Kind = "diagram"
	first.Links = ArtifactLinks{WorkspaceID: "w1", ProjectID: "p1", ChannelID: "c1", CollaborationID: "co1"}
	first.Capabilities = []string{"zoom", "export"}
	second := sampleArtifact("a")
	second.Kind = "document"
	second.Renderer.ID = "text"
	second.Links.WorkspaceID = "w2"
	second.Capabilities = []string{"read"}
	for _, artifact := range []Artifact{first, second} {
		if _, err := store.Create(artifact); err != nil {
			t.Fatal(err)
		}
	}

	all, err := store.List(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "a" || all[1].ID != "b" {
		t.Fatalf("list not sorted: %#v", all)
	}
	filters := []Filter{
		{Kind: "diagram"},
		{WorkspaceID: "w1"},
		{ProjectID: "p1"},
		{ChannelID: "c1"},
		{CollaborationID: "co1"},
		{RendererID: "canvas"},
		{Capability: "export"},
	}
	for _, filter := range filters {
		got, err := store.List(filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != "b" {
			t.Fatalf("filter %#v returned %#v", filter, got)
		}
	}
	none, err := store.List(Filter{Capability: "missing"})
	if err != nil || len(none) != 0 {
		t.Fatalf("missing capability returned %#v, %v", none, err)
	}
}

func TestValidationAndTraversalProtection(t *testing.T) {
	store, err := NewStore(t.TempDir(), WithMaxPayloadBytes(16), WithMaxAssetBytes(4))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../escape", "/absolute", ".", "..", "a/b", `a\b`, ""} {
		if id == "" {
			continue // A blank Create ID intentionally requests generation.
		}
		artifact := sampleArtifact(id)
		if _, err := store.Create(artifact); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("Create(%q) error = %v, want ErrInvalidID", id, err)
		}
		if _, err := store.Get(id); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("Get(%q) error = %v, want ErrInvalidID", id, err)
		}
	}

	invalidJSON := sampleArtifact("bad-json")
	invalidJSON.Payload = json.RawMessage(`{`)
	if _, err := store.Create(invalidJSON); err == nil {
		t.Fatal("invalid JSON payload was accepted")
	}
	missingRenderer := sampleArtifact("bad-renderer")
	missingRenderer.Renderer.ID = ""
	if _, err := store.Create(missingRenderer); err == nil {
		t.Fatal("missing renderer was accepted")
	}
	tooLarge := sampleArtifact("large")
	tooLarge.Payload = json.RawMessage(`{"0123456789":1}`)
	if _, err := store.Create(tooLarge); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("large payload error = %v, want ErrTooLarge", err)
	}
}

func TestGeneratedIDAndDuplicateWithAssets(t *testing.T) {
	store := newTestStore(t)
	source := sampleArtifact("")
	created, err := store.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if validateID(created.ID) != nil {
		t.Fatalf("generated unsafe ID: %q", created.ID)
	}
	if err := store.PutAsset(created.ID, "preview.png", []byte("png")); err != nil {
		t.Fatal(err)
	}

	duplicate, err := store.Duplicate(created.ID, "copy")
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != "copy" || duplicate.Revision != 1 {
		t.Fatalf("unexpected duplicate: %#v", duplicate)
	}
	if len(duplicate.Provenance) == 0 {
		t.Fatal("duplicate missing source provenance")
	}
	ref := duplicate.Provenance[len(duplicate.Provenance)-1]
	if ref.ArtifactID != created.ID || ref.Revision != created.Revision {
		t.Fatalf("unexpected duplicate provenance: %#v", ref)
	}
	asset, err := store.GetAsset("copy", "preview.png")
	if err != nil || string(asset) != "png" {
		t.Fatalf("duplicate asset = %q, %v", asset, err)
	}
}

func TestAssetsAndLimits(t *testing.T) {
	store, err := NewStore(t.TempDir(), WithMaxAssetBytes(4))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(sampleArtifact("assets")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../x", "a/b", ".", "..", ""} {
		if err := store.PutAsset("assets", name, []byte("x")); !errors.Is(err, ErrInvalidAsset) {
			t.Fatalf("PutAsset(%q) error = %v, want ErrInvalidAsset", name, err)
		}
	}
	if err := store.PutAsset("assets", "large.bin", []byte("12345")); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("large asset error = %v, want ErrTooLarge", err)
	}
	if err := store.PutAsset("assets", "ok.bin", []byte("1234")); err != nil {
		t.Fatal(err)
	}
	data, err := store.GetAsset("assets", "ok.bin")
	if err != nil || string(data) != "1234" {
		t.Fatalf("GetAsset = %q, %v", data, err)
	}
	if _, err := store.GetAsset("assets", "missing.bin"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing asset error = %v, want ErrNotFound", err)
	}

	path := filepath.Join(store.Root(), "assets", "assets", "oversized.bin")
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAsset("assets", "oversized.bin"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversized read error = %v, want ErrTooLarge", err)
	}
}

func TestCorruptionSafeReads(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create(sampleArtifact("corrupt")); err != nil {
		t.Fatal(err)
	}
	currentPath := filepath.Join(store.Root(), "corrupt", "artifact.json")
	if err := os.WriteFile(currentPath, []byte(`{"broken":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("corrupt"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("corrupt Get error = %v, want ErrCorrupt", err)
	}
	if _, err := store.List(Filter{}); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("corrupt List error = %v, want ErrCorrupt", err)
	}

	store = newTestStore(t)
	if _, err := store.Create(sampleArtifact("identity")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root(), "identity", "artifact.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(string(data) + `{}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("identity"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("trailing JSON error = %v, want ErrCorrupt", err)
	}
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	created, err := store.Create(sampleArtifact("delete"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("delete", created.Revision+1); !errors.Is(err, ErrConflict) {
		t.Fatalf("Delete conflict error = %v", err)
	}
	if _, err := store.Get("delete"); err != nil {
		t.Fatalf("artifact removed after conflict: %v", err)
	}
	if err := store.Delete("delete", created.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("delete"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
}

func TestStoreOptions(t *testing.T) {
	if _, err := NewStore(t.TempDir(), WithMaxPayloadBytes(0)); err == nil {
		t.Fatal("zero payload limit accepted")
	}
	if _, err := NewStore(t.TempDir(), WithMaxAssetBytes(-1)); err == nil {
		t.Fatal("negative asset limit accepted")
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func sampleArtifact(id string) Artifact {
	return Artifact{
		ID:          id,
		Kind:        "diagram",
		Title:       "Example",
		Description: "test artifact",
		Provenance: []SourceReference{{
			Kind: "prompt", URI: "conversation://one", Label: "request",
			Metadata: map[string]string{"agent": "test"},
		}},
		Links: ArtifactLinks{
			WorkspaceID: "workspace", ProjectID: "project",
			ChannelID: "channel", CollaborationID: "collaboration",
		},
		Renderer:     Renderer{ID: "canvas", APIVersion: "v1", MediaType: "application/vnd.neural-canvas+json"},
		Payload:      json.RawMessage(`{"nodes":[1]}`),
		Fallback:     &Fallback{MediaType: "text/plain", Data: json.RawMessage(`"fallback"`)},
		Capabilities: []string{"zoom", "export"},
	}
}
