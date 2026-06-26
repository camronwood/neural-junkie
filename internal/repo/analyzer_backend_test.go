package repo

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

type mockIndexBackend struct {
	files map[string]string
}

func (m *mockIndexBackend) Kind() string  { return workspacebackend.KindLocal }
func (m *mockIndexBackend) Root() string  { return "/mock" }
func (m *mockIndexBackend) ReadDir(ctx context.Context, rel string) ([]workspacebackend.Entry, error) {
	var out []workspacebackend.Entry
	for path := range m.files {
		if rel == "." || rel == "" {
			out = append(out, workspacebackend.Entry{Name: filepath.Base(path), Path: path, IsDir: false})
		}
	}
	return out, nil
}
func (m *mockIndexBackend) ReadFile(ctx context.Context, rel string) ([]byte, error) {
	if b, ok := m.files[rel]; ok {
		return []byte(b), nil
	}
	return nil, context.Canceled
}
func (m *mockIndexBackend) WriteFile(ctx context.Context, rel string, data []byte) error { return nil }
func (m *mockIndexBackend) Stat(ctx context.Context, rel string) (fs.FileInfo, error) {
	return mockFileInfo{name: rel, mod: time.Now()}, nil
}

type mockFileInfo struct {
	name string
	mod  time.Time
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() fs.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return m.mod }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() interface{}   { return nil }
func (m *mockIndexBackend) Exec(ctx context.Context, req workspacebackend.ExecRequest) (workspacebackend.ExecResult, error) {
	return workspacebackend.ExecResult{}, nil
}

func TestAnalyzeViaBackend_Smoke(t *testing.T) {
	b := &mockIndexBackend{files: map[string]string{
		"README.md": "# Hello",
		"main.go":   "package main\nfunc main() {}",
	}}
	a := NewAnalyzer(nil)
	index, err := a.AnalyzeViaBackend(context.Background(), "/mock", b)
	if err != nil {
		t.Fatal(err)
	}
	if index == nil || len(index.KeyFiles) == 0 {
		t.Fatal("expected key files in index")
	}
	if !strings.Contains(index.ArchitectureDoc, "Hello") && len(index.SourceFiles) == 0 {
		t.Fatal("expected indexed content")
	}
}
