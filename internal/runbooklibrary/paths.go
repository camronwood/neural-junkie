package runbooklibrary

import (
	"os"
	"path/filepath"
)

// UserLibraryDir returns ~/.neural-junkie/runbook-library.
func UserLibraryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "runbook-library"), nil
}

func definitionDir(root, id string) string {
	return filepath.Join(root, sanitizeID(id))
}

func versionPath(root, id string, version int) string {
	return filepath.Join(definitionDir(root, id), versionFilename(version))
}

func manifestPath(root, id string) string {
	return filepath.Join(definitionDir(root, id), "manifest.json")
}

func versionFilename(version int) string {
	if version < 1 {
		version = 1
	}
	return "v" + itoa(version) + ".json"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func sanitizeID(id string) string {
	out := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		} else if c == ' ' || c == '/' {
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return "runbook"
	}
	return string(out)
}
