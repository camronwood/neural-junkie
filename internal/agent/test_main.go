package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain isolates test data from the developer's real ~/.neural-junkie tree.
// Repo agent tests (e.g. widget-expert) index into NEURAL_JUNKIE_REPO_DIR.
func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "neural-junkie-agent-test-home-*")
	if err != nil {
		panic(err)
	}

	_ = os.Setenv("HOME", tmpHome)
	_ = os.Setenv("USERPROFILE", tmpHome)
	_ = os.Setenv("NEURAL_JUNKIE_REPO_DIR", filepath.Join(tmpHome, "repos"))
	_ = os.Setenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR", filepath.Join(tmpHome, "collaborations"))

	code := m.Run()
	_ = os.RemoveAll(tmpHome)
	os.Exit(code)
}
