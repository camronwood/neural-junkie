package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// IsSafeCLIWorkDir reports whether workDir is usable as a CLI agent subprocess
// working directory. Filesystem roots (e.g. "/" from a macOS app launch) are
// rejected because tools like gemini-cli recursively scan the tree and hit
// EPERM on paths such as /Library/Bluetooth.
func IsSafeCLIWorkDir(workDir string) bool {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return false
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return false
	}
	cleaned := filepath.Clean(abs)
	if cleaned == string(filepath.Separator) || cleaned == "." {
		return false
	}
	if runtime.GOOS == "windows" {
		// Volume root like "C:\" or "C:/"
		if len(cleaned) == 3 && cleaned[1] == ':' && (cleaned[2] == '\\' || cleaned[2] == '/') {
			return false
		}
	}
	return true
}

// DefaultCLIWorkDir returns a safe subprocess working directory for CLI agents.
// Prefers process cwd when it is not a filesystem root; otherwise falls back to
// ~/.neural-junkie (created if needed) or the user home directory.
func DefaultCLIWorkDir() string {
	if wd, err := os.Getwd(); err == nil && IsSafeCLIWorkDir(wd) {
		return wd
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		nj := filepath.Join(home, ".neural-junkie")
		if err := os.MkdirAll(nj, 0o755); err == nil {
			return nj
		}
		if IsSafeCLIWorkDir(home) {
			return home
		}
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return wd
	}
	return "."
}
