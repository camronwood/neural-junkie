package music

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := ExpandHomePath("~/.neural-junkie/music/output")
	want := filepath.Join(home, ".neural-junkie", "music", "output")
	if got != want {
		t.Fatalf("ExpandHomePath = %q, want %q", got, want)
	}
}

func TestACEStepStatusFromSettingsNotReady(t *testing.T) {
	dir := t.TempDir()
	settings := map[string]string{
		"ace_step_venv":       filepath.Join(dir, "missing-venv"),
		"ace_step_project":    filepath.Join(dir, "missing-project"),
		"ace_step_checkpoint": filepath.Join(dir, "missing-checkpoint"),
		"music_output_dir":    "~/.neural-junkie/music/output",
	}
	st := ACEStepStatusFromSettings(settings, dir)
	if st.Ready {
		t.Fatal("expected not ready")
	}
	if st.Paths.SetupScript != filepath.Join(dir, "scripts", "setup-acestep.sh") {
		t.Fatalf("setup script path = %q", st.Paths.SetupScript)
	}
}

func TestACEStepStatusFromSettingsReady(t *testing.T) {
	dir := t.TempDir()
	venv := filepath.Join(dir, "venv")
	project := filepath.Join(dir, "project")
	checkpoint := filepath.Join(dir, "checkpoint")
	if err := os.MkdirAll(filepath.Join(venv, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venv, "bin", "python"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(checkpoint, 0o755); err != nil {
		t.Fatal(err)
	}
	st := ACEStepStatusFromSettings(map[string]string{
		"ace_step_venv":       venv,
		"ace_step_project":    project,
		"ace_step_checkpoint": checkpoint,
	}, dir)
	if !st.Ready {
		t.Fatalf("expected ready, got %+v", st)
	}
}
