package workspacebackend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// RemoteBackend talks to an nj-remote sidecar over HTTP.
type RemoteBackend struct {
	root      string
	sidecarURL string
	token     string
	client    *http.Client
}

// NewRemote creates a remote workspace backend.
func NewRemote(root, sidecarURL, token string) *RemoteBackend {
	return &RemoteBackend{
		root:       root,
		sidecarURL: strings.TrimRight(sidecarURL, "/"),
		token:      token,
		client:     &http.Client{Timeout: 120 * time.Second},
	}
}

func (b *RemoteBackend) Kind() string { return KindSSH }

func (b *RemoteBackend) Root() string { return b.root }

func (b *RemoteBackend) do(ctx context.Context, method, path string, body io.Reader, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, b.sidecarURL+path, body)
	if err != nil {
		return err
	}
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sidecar %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (b *RemoteBackend) ReadDir(ctx context.Context, rel string) ([]Entry, error) {
	var payload struct {
		Entries []Entry `json:"entries"`
	}
	path := "/api/fs/list?path=" + strings.TrimPrefix(rel, "/")
	if err := b.do(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Entries, nil
}

func (b *RemoteBackend) ReadFile(ctx context.Context, rel string) ([]byte, error) {
	var payload struct {
		Content string `json:"content"`
	}
	path := "/api/fs/read?path=" + strings.TrimPrefix(rel, "/")
	if err := b.do(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return nil, err
	}
	return []byte(payload.Content), nil
}

func (b *RemoteBackend) WriteFile(ctx context.Context, rel string, data []byte) error {
	body, _ := json.Marshal(map[string]string{
		"path":    strings.TrimPrefix(rel, "/"),
		"content": string(data),
	})
	return b.do(ctx, http.MethodPost, "/api/fs/write", bytes.NewReader(body), nil)
}

func (b *RemoteBackend) Stat(ctx context.Context, rel string) (fs.FileInfo, error) {
	var payload struct {
		Name    string `json:"name"`
		Size    int64  `json:"size"`
		IsDir   bool   `json:"is_dir"`
		ModTime string `json:"mod_time"`
	}
	path := "/api/fs/stat?path=" + strings.TrimPrefix(rel, "/")
	if err := b.do(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return fsFileInfo{}, err
	}
	t, _ := time.Parse(time.RFC3339, payload.ModTime)
	return fsFileInfo{name: payload.Name, size: payload.Size, isDir: payload.IsDir, mod: t}, nil
}

func (b *RemoteBackend) Exec(ctx context.Context, req ExecRequest) (ExecResult, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"command": req.Command,
		"args":    req.Args,
		"cwd":     req.RelCwd,
		"env":     req.Env,
	})
	var res ExecResult
	if err := b.do(ctx, http.MethodPost, "/api/exec", bytes.NewReader(body), &res); err != nil {
		return ExecResult{}, err
	}
	return res, nil
}

// fsFileInfo is a minimal os.FileInfo for remote stat.
type fsFileInfo struct {
	name  string
	size  int64
	isDir bool
	mod   time.Time
}

func (f fsFileInfo) Name() string       { return f.name }
func (f fsFileInfo) Size() int64        { return f.size }
func (f fsFileInfo) Mode() fs.FileMode {
	if f.isDir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (f fsFileInfo) ModTime() time.Time { return f.mod }
func (f fsFileInfo) IsDir() bool        { return f.isDir }
func (f fsFileInfo) Sys() interface{}   { return nil }
