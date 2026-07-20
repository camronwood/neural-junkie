package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/codeindex/graph"
)

func TestHandleRepoGraph_requiresRepoPath(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repo/graph", nil)
	handleRepoGraph(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleRepoGraphSubgraph_requiresQuery(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repo/graph/subgraph?repo_path=/tmp/x", nil)
	handleRepoGraphSubgraph(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleRepoGraph_methodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repo/graph?repo_path=/tmp/x", nil)
	handleRepoGraph(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandleRepoGraph_returnsSummaryWhenReady(t *testing.T) {
	repo := t.TempDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".neural-junkie", "code-graph", graph.RepoHash(repo))
	store, err := graph.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		_ = os.RemoveAll(dir)
	})
	nodes := []graph.Node{
		{ID: "pkg:root", Kind: graph.NodePackage, Label: "root", Community: "root", Degree: 1},
	}
	if err := store.ReplaceAll(nodes, nil, graph.Meta{
		RepoPath: repo, RepoHash: graph.RepoHash(repo), Ready: true, NodeCount: 1,
	}); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repo/graph?repo_path="+url.QueryEscape(repo), nil)
	handleRepoGraph(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var summary graph.GraphSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if !summary.Meta.Ready {
		t.Fatalf("expected ready summary, got %+v", summary.Meta)
	}
}
