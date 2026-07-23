package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsSafeCLIWorkDir(t *testing.T) {
	t.Parallel()
	if IsSafeCLIWorkDir("") {
		t.Fatal("empty should be unsafe")
	}
	if IsSafeCLIWorkDir("/") {
		t.Fatal("filesystem root should be unsafe")
	}
	if IsSafeCLIWorkDir("/tmp/project") != true {
		t.Fatal("normal path should be safe")
	}
	if runtime.GOOS == "windows" {
		if IsSafeCLIWorkDir(`C:\`) {
			t.Fatal("windows volume root should be unsafe")
		}
	}
}

func TestDefaultCLIWorkDir_prefersSafeCwd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got := DefaultCLIWorkDir()
	if IsSafeCLIWorkDir(cwd) {
		if got != cwd {
			t.Fatalf("got %q, want cwd %q", got, cwd)
		}
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".neural-junkie")
	if got != want {
		t.Fatalf("got %q, want %q when cwd unsafe", got, want)
	}
}
