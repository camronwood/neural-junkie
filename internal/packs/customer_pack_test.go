package packs_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func TestInstallFromZipBytesCustomerPack(t *testing.T) {
	src := filepath.Join("testdata", "brightest-bio-lab")
	if _, err := os.Stat(filepath.Join(src, "pack.yaml")); err != nil {
		t.Skip("brightest-bio-lab fixture missing")
	}
	zipPath := filepath.Join(t.TempDir(), "pack.zip")
	if err := zipDir(src, zipPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	m, err := packs.InstallFromZipBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "brightest-bio-lab" {
		t.Fatalf("id: got %q", m.ID)
	}
	if !m.IsCustomerPack() {
		t.Fatal("expected customer pack")
	}
	guide, err := packs.ReadWorkspaceGuide(m, filepath.Join(home, ".neural-junkie", "packs", m.ID))
	if err != nil {
		t.Fatal(err)
	}
	if guide == "" || !contains(guide, "Life sciences") {
		t.Fatal("expected workspace guide content")
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

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
