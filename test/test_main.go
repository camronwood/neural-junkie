package test

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain isolates test data from the developer's real ~/.neural-junkie data.
func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "neural-junkie-test-home-*")
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
