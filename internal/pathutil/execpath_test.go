package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookPathIn(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-cli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := LookPathIn("fake-cli", dir)
	if err != nil {
		t.Fatal(err)
	}
	if path != bin {
		t.Fatalf("got %q want %q", path, bin)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	got := ExpandHome("~/foo")
	want := filepath.Join(home, "foo")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
