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

func TestInstallFromZipBytesRejectsPathLikeAndCaseOfficialIDs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []struct {
		name string
		id   string
		want string
	}{
		{"dot-slash-official", "./life-sciences", "relative path"},
		{"case-official", "Life-Sciences", "official catalog pack"},
		{"traversal", "../victim", "relative path"},
		{"dot-prefix", ".hidden", "relative path"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			packDir := filepath.Join(dir, "pack")
			if err := os.MkdirAll(packDir, 0o755); err != nil {
				t.Fatal(err)
			}
			yaml := "id: " + tc.id + "\nversion: 1.0.0\ntitle: Fake\ndescription: x\n"
			if err := os.WriteFile(filepath.Join(packDir, "pack.yaml"), []byte(yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			zipPath := filepath.Join(dir, "p.zip")
			if err := zipDir(packDir, zipPath); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(zipPath)
			if err != nil {
				t.Fatal(err)
			}
			_, err = packs.InstallFromZipBytes(data)
			if err == nil {
				t.Fatal("expected reject")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) &&
				!strings.Contains(err.Error(), tc.want) {
				// official catalog uses exact phrase; path uses relative path
				if tc.want == "official catalog pack" && strings.Contains(err.Error(), "official catalog pack") {
					return
				}
				if tc.want == "relative path" && (strings.Contains(err.Error(), "relative path") || strings.Contains(err.Error(), "must match")) {
					return
				}
				t.Fatalf("error %v does not contain %q", err, tc.want)
			}
		})
	}
}

func TestNormalizeAndOfficialPackIDCaseInsensitive(t *testing.T) {
	if !packs.IsOfficialPackID("Life-Sciences") {
		t.Fatal("expected case-insensitive official match")
	}
	if err := packs.ValidatePackIDFormat("../x"); err == nil {
		t.Fatal("expected path reject")
	}
	if err := packs.ValidatePackIDFormat("./life-sciences"); err == nil {
		t.Fatal("expected ./ reject")
	}
	dest, err := packs.PackInstallDest(t.TempDir(), "my-pack")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(dest, string(filepath.Separator)+"my-pack") && filepath.Base(dest) != "my-pack" {
		t.Fatalf("unexpected dest %s", dest)
	}
}
