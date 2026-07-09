package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func TestListPackUpdatesDetectsNewerCatalogVersion(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackMusicCreation)

	cat, err := packs.FetchCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var dlURL string
	for _, e := range cat.Packs {
		if e.ID == PackMusicCreation {
			dlURL = e.DownloadURL
			break
		}
	}
	if dlURL == "" {
		t.Fatal("missing music-creation download url")
	}

	packs.InvalidateCatalogCache()
	body, err := json.Marshal(&packs.Catalog{
		Version: 1,
		Packs: []packs.CatalogEntry{{
			ID:          PackMusicCreation,
			Version:     "2.0.1",
			Title:       "Music creation",
			DownloadURL: dlURL,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	t.Setenv("NEURAL_JUNKIE_PACKS_CATALOG_URL", srv.URL)

	updates, err := cfg.ListPackUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].ID != PackMusicCreation {
		t.Fatalf("updates = %+v", updates)
	}
	if updates[0].InstalledVersion == "" || updates[0].LatestVersion != "2.0.1" {
		t.Fatalf("unexpected versions: %+v", updates[0])
	}

	rows, err := cfg.ListPackCatalogStatus()
	if err != nil {
		t.Fatal(err)
	}
	var row *PackCatalogStatus
	for i := range rows {
		if rows[i].ID == PackMusicCreation {
			row = &rows[i]
			break
		}
	}
	if row == nil || !row.UpdateAvailable || row.Version != "2.0.1" {
		t.Fatalf("catalog row = %+v", row)
	}
}

func TestUpgradePackPreservesEnabled(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackMusicCreation)
	if err := cfg.SetPackEnabled(PackMusicCreation, true); err != nil {
		t.Fatal(err)
	}

	upgradedZipURL := serveTestPackZipAtVersion(t, PackMusicCreation, "2.0.1")
	packs.InvalidateCatalogCache()
	body, _ := json.Marshal(&packs.Catalog{
		Version: 1,
		Packs: []packs.CatalogEntry{{
			ID: PackMusicCreation, Version: "2.0.1", Title: "Music", DownloadURL: upgradedZipURL,
		}},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	t.Setenv("NEURAL_JUNKIE_PACKS_CATALOG_URL", srv.URL)

	wasEnabled, err := cfg.UpgradePack(PackMusicCreation)
	if err != nil {
		t.Fatal(err)
	}
	if !wasEnabled || !cfg.IsPackEnabled(PackMusicCreation) {
		t.Fatal("expected pack to stay enabled after upgrade")
	}
	if !cfg.IsPackInstalled(PackMusicCreation) {
		t.Fatal("expected pack still installed")
	}
	ver := cfg.installedCatalogVersion(PackMusicCreation)
	if ver != "2.0.1" {
		t.Fatalf("installed version = %q, want 2.0.1", ver)
	}
}

// serveTestPackZipAtVersion zips an official fixture with pack.yaml version overridden.
func serveTestPackZipAtVersion(t *testing.T, packID, version string) string {
	t.Helper()
	src, err := officialPackFixtureDir(packID)
	if err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	if err := copyDirForTest(src, stage); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(stage, "pack.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "version:") {
			lines[i] = fmt.Sprintf("version: %q", version)
			break
		}
	}
	if err := os.WriteFile(manifestPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), packID+".zip")
	if err := zipDirContents(stage, zipPath); err != nil {
		t.Fatal(err)
	}
	zipSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	t.Cleanup(zipSrv.Close)
	restoreClient := packs.SetInstallHTTPClientForTests(zipSrv.Client())
	t.Cleanup(restoreClient)
	return zipSrv.URL + "/" + packID + ".zip"
}

func copyDirForTest(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode()&0777|0600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
