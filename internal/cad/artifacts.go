package cad

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultProjectFile = "model.scad"
	defaultPreviewSTL  = "preview.stl"
)

// ProjectPaths holds canonical paths within a CAD project directory.
type ProjectPaths struct {
	Root      string
	SCADPath  string
	STLPath   string
	VersionsDir string
}

// ResolveArtifactsRoot expands and creates the configured artifacts root.
func ResolveArtifactsRoot(dir string) (string, error) {
	dir = expandHome(strings.TrimSpace(dir))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".neural-junkie", "cad")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ProjectDir returns (and creates) ~/.neural-junkie/cad/<projectID>/.
func ProjectDir(artifactsRoot, projectID string) (ProjectPaths, error) {
	root, err := ResolveArtifactsRoot(artifactsRoot)
	if err != nil {
		return ProjectPaths{}, err
	}
	projectID = sanitizeProjectID(projectID)
	if projectID == "" {
		projectID = "default"
	}
	dir := filepath.Join(root, projectID)
	if err := os.MkdirAll(filepath.Join(dir, "versions"), 0755); err != nil {
		return ProjectPaths{}, err
	}
	return ProjectPaths{
		Root:        dir,
		SCADPath:    filepath.Join(dir, defaultProjectFile),
		STLPath:     filepath.Join(dir, defaultPreviewSTL),
		VersionsDir: filepath.Join(dir, "versions"),
	}, nil
}

func sanitizeProjectID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == '/' || r == '\\' || r == ' ' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

// WriteSCADFile writes content to path, creating parent dirs.
func WriteSCADFile(path, content string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadSCADFile reads a .scad file.
func ReadSCADFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CopyFile copies src to dst.
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
