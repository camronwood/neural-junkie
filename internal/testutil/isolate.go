package testutil

import (
	"path/filepath"
	"testing"
)

// IsolateNeuralJunkieHome redirects neural-junkie data to a temp tree for the test,
// preventing artifacts from polluting ~/.neural-junkie.
func IsolateNeuralJunkieHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NEURAL_JUNKIE_REPO_DIR", filepath.Join(home, "repos"))
	t.Setenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR", filepath.Join(home, "collaborations"))
	return home
}
