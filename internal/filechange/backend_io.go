package filechange

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// WorkspaceIO abstracts filesystem ops for local and remote workspaces.
type WorkspaceIO interface {
	Root() string
	ReadFile(ctx context.Context, rel string) ([]byte, error)
	WriteFile(ctx context.Context, rel string, data []byte) error
	Stat(ctx context.Context, rel string) (fs.FileInfo, error)
	Exec(ctx context.Context, req workspacebackend.ExecRequest) (workspacebackend.ExecResult, error)
}

// BackendIO adapts workspacebackend.Backend for file change execution.
type BackendIO struct {
	Backend workspacebackend.Backend
}

func (b BackendIO) Root() string { return b.Backend.Root() }

func (b BackendIO) ReadFile(ctx context.Context, rel string) ([]byte, error) {
	return b.Backend.ReadFile(ctx, rel)
}

func (b BackendIO) WriteFile(ctx context.Context, rel string, data []byte) error {
	return b.Backend.WriteFile(ctx, rel, data)
}

func (b BackendIO) Stat(ctx context.Context, rel string) (fs.FileInfo, error) {
	return b.Backend.Stat(ctx, rel)
}

func (b BackendIO) Exec(ctx context.Context, req workspacebackend.ExecRequest) (workspacebackend.ExecResult, error) {
	return b.Backend.Exec(ctx, req)
}

func (fce *FileChangeExecutor) absToRel(absPath string) (string, error) {
	root := filepath.Clean(fce.workspaceRoot)
	candidate := filepath.Clean(absPath)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path outside workspace: %s", absPath)
	}
	return rel, nil
}

func (fce *FileChangeExecutor) ioRead(absPath string) ([]byte, error) {
	if fce.workspaceIO != nil {
		rel, err := fce.absToRel(absPath)
		if err != nil {
			return nil, err
		}
		return fce.workspaceIO.ReadFile(context.Background(), rel)
	}
	return os.ReadFile(absPath)
}

func (fce *FileChangeExecutor) ioWrite(absPath string, data []byte) error {
	if fce.workspaceIO != nil {
		rel, err := fce.absToRel(absPath)
		if err != nil {
			return err
		}
		return fce.workspaceIO.WriteFile(context.Background(), rel, data)
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(absPath, data, 0o644)
}

func (fce *FileChangeExecutor) ioStat(absPath string) (fs.FileInfo, error) {
	if fce.workspaceIO != nil {
		rel, err := fce.absToRel(absPath)
		if err != nil {
			return nil, err
		}
		return fce.workspaceIO.Stat(context.Background(), rel)
	}
	return os.Stat(absPath)
}

func (fce *FileChangeExecutor) ioRemove(absPath string) error {
	if fce.workspaceIO != nil {
		rel, err := fce.absToRel(absPath)
		if err != nil {
			return err
		}
		res, err := fce.workspaceIO.Exec(context.Background(), workspacebackend.ExecRequest{
			Command: "rm",
			Args:    []string{"-f", rel},
			RelCwd:  ".",
		})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("rm: %s", strings.TrimSpace(res.Stderr))
		}
		return nil
	}
	return os.Remove(absPath)
}

func (fce *FileChangeExecutor) ioRename(oldAbs, newAbs string) error {
	if fce.workspaceIO != nil {
		oldRel, err := fce.absToRel(oldAbs)
		if err != nil {
			return err
		}
		newRel, err := fce.absToRel(newAbs)
		if err != nil {
			return err
		}
		res, err := fce.workspaceIO.Exec(context.Background(), workspacebackend.ExecRequest{
			Command: "mv",
			Args:    []string{oldRel, newRel},
			RelCwd:  ".",
		})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("mv: %s", strings.TrimSpace(res.Stderr))
		}
		return nil
	}
	destDir := filepath.Dir(newAbs)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return os.Rename(oldAbs, newAbs)
}
