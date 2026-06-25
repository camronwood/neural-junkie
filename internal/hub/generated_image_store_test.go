package hub

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveGeneratedImageFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	path, err := saveGeneratedImageFile("msg-abc", "image/png", base64.StdEncoding.EncodeToString([]byte("png-bytes")))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "png-bytes" {
		t.Fatalf("file contents = %q", data)
	}
	wantDir := filepath.Join(tmpHome, ".neural-junkie", generatedImagesDirName)
	if filepath.Dir(path) != wantDir {
		t.Fatalf("dir = %q want %q", filepath.Dir(path), wantDir)
	}
}
