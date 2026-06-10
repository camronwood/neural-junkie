package learning

import (
	"os"
	"testing"
)

// testDataDir returns an isolated directory; callers must defer unlock() before RemoveAll.
func testDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "nj-learning-*")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
