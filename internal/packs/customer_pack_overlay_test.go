package packs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSettingsOverlayExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	packDir := t.TempDir()
	m := &Manifest{
		SettingsOverlay: map[string]string{
			"music_output_dir":    "~/.neural-junkie/music/output",
			"ace_step_venv":       "~/.neural-junkie/music/acestep-venv",
			"ace_step_checkpoint": "~/.neural-junkie/music/checkpoints/acestep-v15-sft",
		},
	}
	got, err := ResolveSettingsOverlay(m, packDir)
	if err != nil {
		t.Fatal(err)
	}
	wantOut := filepath.Join(home, ".neural-junkie", "music", "output")
	if got["music_output_dir"] != wantOut {
		t.Fatalf("music_output_dir = %q, want %q", got["music_output_dir"], wantOut)
	}
	wantVenv := filepath.Join(home, ".neural-junkie", "music", "acestep-venv")
	if got["ace_step_venv"] != wantVenv {
		t.Fatalf("ace_step_venv = %q, want %q", got["ace_step_venv"], wantVenv)
	}
}
