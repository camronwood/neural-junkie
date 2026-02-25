package test

import (
	"os"
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

	code := m.Run()
	_ = os.RemoveAll(tmpHome)
	os.Exit(code)
}
