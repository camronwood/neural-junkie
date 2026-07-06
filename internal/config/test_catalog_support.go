package config

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// SetupTestOfficialPackCatalog serves local official-pack fixture zips for InstallPack in tests.
func SetupTestOfficialPackCatalog(t *testing.T) {
	t.Helper()
	setupTestOfficialPackCatalog(t)
}

// InstallTestPack syncs an official pack fixture into the test home and updates config.
func InstallTestPack(t *testing.T, cfg *Config, packID string) {
	installTestPack(t, cfg, packID)
}

func officialPackFixtureDir(packID string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrInvalid
	}
	root := filepath.Join(filepath.Dir(file), "..", "packs", "testdata", "official", packID)
	return filepath.Abs(root)
}

func installTestPack(t *testing.T, cfg *Config, packID string) {
	t.Helper()
	syncTestPack(t, packID)
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if cfg.Packs.Enabled == nil {
		cfg.Packs.Enabled = make(map[string]bool)
	}
	if !cfg.packInstalledLocked(packID) {
		cfg.Packs.Installed = append(cfg.Packs.Installed, packID)
	}
	cfg.Packs.Enabled[packID] = false
}

func syncTestPack(t *testing.T, packID string) {
	t.Helper()
	dir, err := officialPackFixtureDir(packID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packs.SyncPackFromDir(dir); err != nil {
		t.Fatal(err)
	}
}

func setupTestOfficialPackCatalog(t *testing.T) {
	t.Helper()
	packs.InvalidateCatalogCache()
	t.Setenv("NEURAL_JUNKIE_PACKS_ALLOW_TEST_HOSTS", "1")

	zipDir := t.TempDir()
	entries := make([]packs.CatalogEntry, 0, len(packs.OfficialPackIDs))
	for _, id := range packs.OfficialPackIDs {
		src, err := officialPackFixtureDir(id)
		if err != nil {
			t.Fatal(err)
		}
		zipPath := filepath.Join(zipDir, id+".zip")
		if err := zipDirContents(src, zipPath); err != nil {
			t.Fatal(err)
		}
		ver := "1.0.0"
		if m, err := packs.LoadManifest(src); err == nil && m != nil && m.Version != "" {
			ver = m.Version
		}
		entries = append(entries, packs.CatalogEntry{
			ID:          id,
			Version:     ver,
			Title:       id,
			Builtin:     true,
			DownloadURL: "",
		})
	}

	zipSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		http.ServeFile(w, r, filepath.Join(zipDir, name))
	}))
	t.Cleanup(zipSrv.Close)
	restoreClient := packs.SetInstallHTTPClientForTests(zipSrv.Client())
	t.Cleanup(restoreClient)

	for i := range entries {
		entries[i].DownloadURL = zipSrv.URL + "/" + entries[i].ID + ".zip"
	}
	catalogBody, err := json.Marshal(&packs.Catalog{Version: 1, Packs: entries})
	if err != nil {
		t.Fatal(err)
	}

	catalogSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(catalogBody)
	}))
	t.Cleanup(func() {
		catalogSrv.Close()
		packs.InvalidateCatalogCache()
	})
	t.Setenv("NEURAL_JUNKIE_PACKS_CATALOG_URL", catalogSrv.URL)
}

func zipDirContents(srcDir, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()
	w := zip.NewWriter(out)
	defer w.Close()
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		fw, err := w.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
}
