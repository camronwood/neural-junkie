package packs

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDownloadURL(t *testing.T) {
	if err := validateDownloadURL("https://github.com/camronwood/neural-junkie/releases/download/packs-v1.0.0/software-development-1.0.0.zip"); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadURL("http://github.com/foo.zip"); err == nil {
		t.Fatal("expected http reject")
	}
	if err := validateDownloadURL("https://evil.example/pack.zip"); err == nil {
		t.Fatal("expected host reject")
	}
}

func TestInstallFromZipURL(t *testing.T) {
	InvalidateCatalogCache()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "pack.yaml")
	if err := os.WriteFile(manifest, []byte(`id: software-development
version: "1.0.0"
title: Software development
`), 0644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "pack.zip")
	if err := zipDir(dir, zipPath); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer srv.Close()
	prevClient := installHTTPClient
	installHTTPClient = srv.Client()
	t.Cleanup(func() { installHTTPClient = prevClient })

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NEURAL_JUNKIE_PACKS_ALLOW_TEST_HOSTS", "1")

	if err := installFromZipURL("software-development", srv.URL, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	installed, err := LoadManifest(filepath.Join(home, ".neural-junkie", "packs", "software-development"))
	if err != nil {
		t.Fatal(err)
	}
	if installed.ID != "software-development" {
		t.Fatalf("got id %q", installed.ID)
	}
}

func zipDir(srcDir, zipPath string) error {
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
