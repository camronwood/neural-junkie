package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCollabAssetsRootDefault(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR", "")

	got, err := ResolveCollabAssetsRoot("")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmpHome, ".neural-junkie", "collaborations")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveCollabAssetsRootTilde(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	custom := filepath.Join(tmpHome, "collab-output")
	got, err := ResolveCollabAssetsRoot("~/collab-output")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(custom) {
		t.Fatalf("got %q want %q", got, custom)
	}
}

func TestCollabAssetsRootEnvOverridesConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	envDir := filepath.Join(tmpHome, "from-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR", envDir)

	cfg := DefaultConfig()
	cfg.Collaboration.AssetsRoot = filepath.Join(tmpHome, "from-config")

	got := CollabAssetsRoot(cfg)
	if filepath.Clean(got) != filepath.Clean(envDir) {
		t.Fatalf("env override: got %q want %q", got, envDir)
	}
}

func TestCollabAssetsRootFromConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR", "")

	cfgDir := filepath.Join(tmpHome, "my-collabs")
	cfg := DefaultConfig()
	cfg.Collaboration.AssetsRoot = cfgDir

	got := CollabAssetsRoot(cfg)
	if filepath.Clean(got) != filepath.Clean(cfgDir) {
		t.Fatalf("got %q want %q", got, cfgDir)
	}
}
