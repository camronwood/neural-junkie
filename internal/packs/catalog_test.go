package packs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchCatalogUsesRemote(t *testing.T) {
	InvalidateCatalogCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":1,"packs":[{"id":"software-development","version":"9.9.9","title":"Remote","download_url":"https://github.com/camronwood/neural-junkie-pack-software-development/releases/download/v1.0.0/software-development-1.0.0.zip"}]}`))
	}))
	defer srv.Close()
	t.Setenv("NEURAL_JUNKIE_PACKS_CATALOG_URL", srv.URL)
	cat, err := FetchCatalog()
	if err != nil {
		t.Fatal(err)
	}
	e := cat.CatalogEntryByID("software-development")
	if e == nil || e.Version != "9.9.9" || e.Title != "Remote" {
		t.Fatalf("unexpected entry: %+v", e)
	}
}

func TestFetchCatalogRequiresRemote(t *testing.T) {
	InvalidateCatalogCache()
	t.Setenv("NEURAL_JUNKIE_PACKS_CATALOG_URL", "http://127.0.0.1:1/unreachable")
	_, err := FetchCatalog()
	if err == nil {
		t.Fatal("expected error when catalog unreachable")
	}
}

func TestMergeBuiltinOfficialPacks(t *testing.T) {
	cat := &Catalog{
		Version: 1,
		Packs: []CatalogEntry{
			{ID: "software-development", Title: "Dev", Version: "1.0.0"},
			{ID: "aws", Title: "AWS", Version: "1.0.0"},
		},
	}
	merged, err := mergeBuiltinOfficialPacks(cat)
	if err != nil {
		t.Fatal(err)
	}
	if merged.CatalogEntryByID("web-browser") == nil {
		t.Fatal("expected embedded web-browser pack to be merged into stale catalog")
	}
}

func TestOrderCatalogPacks(t *testing.T) {
	cat := orderCatalogPacks(&Catalog{
		Version: 1,
		Packs: []CatalogEntry{
			{ID: "cad", Title: "CAD"},
			{ID: "software-development", Title: "Dev"},
			{ID: "custom-pack", Title: "Custom"},
		},
	})
	if len(cat.Packs) != 3 {
		t.Fatalf("got %d packs", len(cat.Packs))
	}
	if cat.Packs[0].ID != "software-development" || cat.Packs[2].ID != "custom-pack" {
		t.Fatalf("unexpected order: %+v", cat.Packs)
	}
}

func TestCatalogEmbedMatchesRepoCatalog(t *testing.T) {
	repoPath := filepath.Join("..", "..", "packs", "catalog.json")
	repoBytes, err := os.ReadFile(repoPath)
	if err != nil {
		t.Fatalf("read packs/catalog.json: %v", err)
	}
	var repo Catalog
	if err := json.Unmarshal(repoBytes, &repo); err != nil {
		t.Fatalf("parse packs/catalog.json: %v", err)
	}
	embed, err := builtinOfficialCatalog()
	if err != nil {
		t.Fatal(err)
	}
	repoByID := map[string]CatalogEntry{}
	for _, e := range repo.Packs {
		repoByID[e.ID] = e
	}
	embedByID := map[string]CatalogEntry{}
	for _, e := range embed.Packs {
		embedByID[e.ID] = e
	}
	if len(repoByID) != len(embedByID) {
		t.Fatalf("catalog pack count drift: repo=%d embed=%d", len(repoByID), len(embedByID))
	}
	for id, re := range repoByID {
		ee, ok := embedByID[id]
		if !ok {
			t.Errorf("pack %q in packs/catalog.json missing from official_catalog.json", id)
			continue
		}
		if re.Version != ee.Version {
			t.Errorf("pack %q version drift: repo=%s embed=%s", id, re.Version, ee.Version)
		}
		if re.DownloadURL != ee.DownloadURL {
			t.Errorf("pack %q download_url drift", id)
		}
	}
	officialSet := map[string]struct{}{}
	for _, id := range OfficialPackIDs {
		officialSet[id] = struct{}{}
	}
	for id := range repoByID {
		if _, ok := officialSet[id]; !ok {
			t.Errorf("pack %q in catalog but missing from OfficialPackIDs", id)
		}
	}
	for id := range officialSet {
		if _, ok := repoByID[id]; !ok {
			t.Errorf("OfficialPackIDs has %q missing from packs/catalog.json", id)
		}
	}
}
