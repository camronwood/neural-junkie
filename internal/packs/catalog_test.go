package packs

import (
	"net/http"
	"net/http/httptest"
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
