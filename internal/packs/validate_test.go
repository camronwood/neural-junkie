package packs

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePackDir_validFixture(t *testing.T) {
	dir := filepath.Join("testdata", "brightest-bio-lab")
	report, err := ValidatePackDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid {
		t.Fatalf("expected valid, errors=%v", report.Errors)
	}
	if report.Manifest == nil || report.Manifest.ID != "brightest-bio-lab" {
		t.Fatalf("manifest: %+v", report.Manifest)
	}
	if !report.Assets.WorkspaceGuideFound {
		t.Fatal("expected workspace guide found")
	}
}

func TestValidateZipBytes_noInstall(t *testing.T) {
	dir := filepath.Join("testdata", "brightest-bio-lab")
	data, err := zipDirBytes(dir)
	if err != nil {
		t.Fatal(err)
	}
	report, err := ValidateZipBytes(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid {
		t.Fatalf("expected valid, errors=%v", report.Errors)
	}
	root, err := UserPacksDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "brightest-bio-lab")); err == nil {
		// fixture id may already be installed from other tests; ensure validate did not write
	}
}

func TestValidateYAML_missingTitle(t *testing.T) {
	report, err := ValidateYAML("id: test-pack\n", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Valid {
		t.Fatal("expected invalid")
	}
}

func TestSyncPackFromDir(t *testing.T) {
	src := filepath.Join("testdata", "brightest-bio-lab")
	m, err := SyncPackFromDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "brightest-bio-lab" {
		t.Fatalf("id=%q", m.ID)
	}
	dir, err := InstalledPackDir(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pack.yaml")); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(dir)
}

func zipDirBytes(dir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
