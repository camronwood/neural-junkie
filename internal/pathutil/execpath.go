package pathutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LookPathIn searches for an executable in pathEnv (or system PATH when empty).
func LookPathIn(name, pathEnv string) (string, error) {
	if pathEnv == "" {
		return exec.LookPath(name)
	}
	if strings.ContainsAny(name, `/\`) {
		return "", exec.ErrNotFound
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, name)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}

// EnhancedPATH augments the current PATH with common CLI install locations.
func EnhancedPATH() string {
	seen := make(map[string]bool)
	var dirs []string

	add := func(d string) {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			return
		}
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}

	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".local", "bin"))
		add(filepath.Join(home, ".cursor", "bin"))
	}

	add("/opt/homebrew/bin")
	add("/usr/local/bin")

	if out, err := exec.Command("npm", "config", "get", "prefix").Output(); err == nil {
		prefix := strings.TrimSpace(string(out))
		if prefix != "" {
			add(filepath.Join(prefix, "bin"))
		}
	}

	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		add(d)
	}

	return strings.Join(dirs, string(os.PathListSeparator))
}

// ExpandHome replaces a leading ~ with the user home directory.
func ExpandHome(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
