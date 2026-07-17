package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildIndexTinyFixture(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal/hub"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainSrc := `package main

import (
	"fmt"
	"example.com/demo/internal/hub"
)

func main() {
	fmt.Println(hub.Name)
}
`
	hubSrc := `package hub

const Name = "hub"
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "internal/hub/hub.go"), []byte(hubSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Isolate graph dir via HOME
	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := BuildIndex(ctx, dir); err != nil {
		t.Fatal(err)
	}
	meta, err := Status(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.Ready || meta.NodeCount == 0 || meta.EdgeCount == 0 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	sum, err := Summary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Nodes) == 0 {
		t.Fatal("expected UI nodes")
	}
	sg, err := QuerySubgraph(dir, "main", 1, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(sg.Nodes) == 0 {
		t.Fatal("expected subgraph hits for main")
	}
}
