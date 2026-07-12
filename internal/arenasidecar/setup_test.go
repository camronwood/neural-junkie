package arenasidecar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSidecarStatusFromSettingsNoVenv(t *testing.T) {
	dir := t.TempDir()
	st := SidecarStatusFromSettings(map[string]string{
		"arena_venv": filepath.Join(dir, "venv"),
	}, dir)
	if st.VenvReady {
		t.Fatal("expected venv not ready")
	}
	if st.Paths.Requirements != filepath.Join(dir, requirementsFile) {
		t.Fatalf("requirements path = %q", st.Paths.Requirements)
	}
}

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	got := ExpandHomePath("~/arena/venv")
	want := filepath.Join(home, "arena", "venv")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
