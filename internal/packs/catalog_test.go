package packs

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMergeCatalogRemoteOverridesDownloadURL(t *testing.T) {
	base, err := LoadBuiltinCatalog()
	if err != nil {
		t.Fatal(err)
	}
	remote := &Catalog{
		Version: 2,
		Packs: []CatalogEntry{{
			ID:          "software-development",
			Version:     "1.0.1",
			DownloadURL: "https://github.com/example/new.zip",
		}},
	}
	merged := mergeCatalogs(base, remote)
	e := merged.CatalogEntryByID("software-development")
	if e == nil {
		t.Fatal("missing entry")
	}
	if e.DownloadURL != "https://github.com/example/new.zip" {
		t.Fatalf("download_url = %q", e.DownloadURL)
	}
	if e.Version != "1.0.1" {
		t.Fatalf("version = %q", e.Version)
	}
	if e.Title == "" {
		t.Fatal("expected title from embedded")
	}
}

func TestFetchCatalogUsesRemote(t *testing.T) {
	InvalidateCatalogCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":1,"packs":[{"id":"software-development","version":"9.9.9","title":"Remote","download_url":"https://github.com/camronwood/neural-junkie/releases/download/packs-v1.0.0/software-development-1.0.0.zip"}]}`))
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
