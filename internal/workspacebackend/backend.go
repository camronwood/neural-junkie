// Package workspacebackend abstracts local and remote workspace filesystem access.
package workspacebackend

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/remotetokens"
)

// Kind identifies the workspace backend implementation.
const (
	KindLocal = "local"
	KindSSH   = "ssh"
	KindDevcontainer = "devcontainer"
)

// Entry is a directory listing item.
type Entry struct {
	Name    string
	Path    string // relative to workspace root
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// ExecRequest runs a command with cwd relative to workspace root.
type ExecRequest struct {
	Command string
	Args    []string
	RelCwd  string
	Env     []string
	Timeout time.Duration
}

// ExecResult captures subprocess output.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Backend provides filesystem and exec access for a workspace.
type Backend interface {
	Kind() string
	Root() string
	ReadDir(ctx context.Context, rel string) ([]Entry, error)
	ReadFile(ctx context.Context, rel string) ([]byte, error)
	WriteFile(ctx context.Context, rel string, data []byte) error
	Stat(ctx context.Context, rel string) (fs.FileInfo, error)
	Exec(ctx context.Context, req ExecRequest) (ExecResult, error)
}

// WorkspaceRecord is the subset of hub workspace fields needed for backend resolution.
type WorkspaceRecord struct {
	Path       string
	Kind       string
	SidecarURL string
}

// WorkspaceSource resolves workspace metadata by ID.
type WorkspaceSource interface {
	GetWorkspace(id string) (WorkspaceRecord, bool)
}

// Resolver returns a Backend for a workspace.
type Resolver struct {
	source  WorkspaceSource
	remotes map[string]Backend
}

// NewResolver creates a backend resolver.
func NewResolver(source WorkspaceSource) *Resolver {
	return &Resolver{source: source, remotes: make(map[string]Backend)}
}

// RegisterRemote attaches a remote backend for a workspace ID (tests / sidecar wiring).
func (r *Resolver) RegisterRemote(workspaceID string, b Backend) {
	if r.remotes == nil {
		r.remotes = make(map[string]Backend)
	}
	r.remotes[workspaceID] = b
}

// ForWorkspace returns the backend for a workspace ID.
func (r *Resolver) ForWorkspace(workspaceID string) (Backend, error) {
	if r == nil || r.source == nil {
		return nil, errors.New("workspace resolver not configured")
	}
	if b, ok := r.remotes[workspaceID]; ok {
		return b, nil
	}
	ws, ok := r.source.GetWorkspace(workspaceID)
	if !ok {
		return nil, errors.New("workspace not found")
	}
	kind := strings.TrimSpace(ws.Kind)
	if kind == "" {
		kind = KindLocal
	}
	switch kind {
	case KindLocal, "":
		return NewLocal(ws.Path), nil
	case KindSSH, KindDevcontainer:
		if b, ok := r.remotes[workspaceID]; ok {
			return b, nil
		}
		if strings.TrimSpace(ws.SidecarURL) != "" {
			token, _ := remotetokens.Get(workspaceID)
			return NewRemote(ws.Path, ws.SidecarURL, token), nil
		}
		return nil, errors.New("remote workspace not connected")
	default:
		return NewLocal(ws.Path), nil
	}
}

// LocalBackend implements Backend on the local filesystem.
type LocalBackend struct {
	root string
}

// NewLocal creates a local filesystem backend.
func NewLocal(root string) *LocalBackend {
	return &LocalBackend{root: filepath.Clean(root)}
}

func (b *LocalBackend) Kind() string { return KindLocal }

func (b *LocalBackend) Root() string { return b.root }

func (b *LocalBackend) abs(rel string) (string, error) {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	full := filepath.Join(b.root, filepath.FromSlash(rel))
	return pathutil.WithinRoot(b.root, full)
}

func (b *LocalBackend) ReadDir(ctx context.Context, rel string) ([]Entry, error) {
	_ = ctx
	abs, err := b.abs(rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	base := strings.TrimPrefix(filepath.ToSlash(rel), "/")
	var out []Entry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		p := e.Name()
		if base != "" && base != "." {
			p = base + "/" + e.Name()
		}
		out = append(out, Entry{
			Name:    e.Name(),
			Path:    p,
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return out, nil
}

func (b *LocalBackend) ReadFile(ctx context.Context, rel string) ([]byte, error) {
	_ = ctx
	abs, err := b.abs(rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

func (b *LocalBackend) WriteFile(ctx context.Context, rel string, data []byte) error {
	_ = ctx
	abs, err := b.abs(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}

func (b *LocalBackend) Stat(ctx context.Context, rel string) (fs.FileInfo, error) {
	_ = ctx
	abs, err := b.abs(rel)
	if err != nil {
		return nil, err
	}
	return os.Stat(abs)
}

func (b *LocalBackend) Exec(ctx context.Context, req ExecRequest) (ExecResult, error) {
	relCwd := req.RelCwd
	if relCwd == "" {
		relCwd = "."
	}
	cwd, err := b.abs(relCwd)
	if err != nil {
		return ExecResult{}, err
	}
	return runLocalCommand(ctx, cwd, req)
}
