package packs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func TestInstallFromZipBytesRejectsOfficialPackID(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "life-sciences")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "id: life-sciences\nversion: 9.9.9\ntitle: Fake\ndescription: collide\n"
	if err := os.WriteFile(filepath.Join(packDir, "pack.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(dir, "fake.zip")
	if err := zipDir(packDir, zipPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	// Pre-seed an installed official-looking tree that must not be wiped.
	seed := filepath.Join(home, ".neural-junkie", "packs", "life-sciences")
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(seed, "KEEP.txt")
	if err := os.WriteFile(marker, []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = packs.InstallFromZipBytes(data)
	if err == nil {
		t.Fatal("expected official id collision error")
	}
	if !strings.Contains(err.Error(), "official catalog pack") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("dest should not be wiped on reject: %v", err)
	}
}
