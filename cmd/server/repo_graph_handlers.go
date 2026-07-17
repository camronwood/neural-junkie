package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/codeindex/graph"
)

func repoPathFromGraphRequest(r *http.Request) string {
	q := strings.TrimSpace(r.URL.Query().Get("repo_path"))
	if q != "" {
		return q
	}
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if wsID != "" && workspaceManager != nil {
		if ws, ok := workspaceManager.GetWorkspace(wsID); ok && ws != nil {
			return ws.Path
		}
	}
	return ""
}

// handleRepoGraph returns communities, god nodes, and a dense UI subgraph.
func handleRepoGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoPath := repoPathFromGraphRequest(r)
	if repoPath == "" {
		http.Error(w, "repo_path required", http.StatusBadRequest)
		return
	}
	meta, _ := graph.Status(repoPath)
	if !meta.Ready && !meta.Building {
		graph.BuildIndexAsync(repoPath)
		meta, _ = graph.Status(repoPath)
	}
	if !meta.Ready {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(graph.GraphSummary{Meta: meta})
		return
	}
	summary, err := graph.Summary(repoPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// handleRepoGraphSubgraph returns a scoped subgraph for a query.
func handleRepoGraphSubgraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoPath := repoPathFromGraphRequest(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if repoPath == "" || q == "" {
		http.Error(w, "repo_path and q required", http.StatusBadRequest)
		return
	}
	hops, _ := strconv.Atoi(r.URL.Query().Get("hops"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	meta, _ := graph.Status(repoPath)
	if !meta.Ready && !meta.Building {
		graph.BuildIndexAsync(repoPath)
	}
	sg, err := graph.QuerySubgraph(repoPath, q, hops, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sg)
}

// handleRepoGraphPath returns the shortest path between two nodes.
func handleRepoGraphPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoPath := repoPathFromGraphRequest(r)
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if repoPath == "" || from == "" || to == "" {
		http.Error(w, "repo_path, from, and to required", http.StatusBadRequest)
		return
	}
	res, err := graph.ShortestPath(repoPath, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleRepoGraphExplain explains a single node.
func handleRepoGraphExplain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoPath := repoPathFromGraphRequest(r)
	node := strings.TrimSpace(r.URL.Query().Get("node"))
	if repoPath == "" || node == "" {
		http.Error(w, "repo_path and node required", http.StatusBadRequest)
		return
	}
	res, err := graph.Explain(repoPath, node)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleRepoGraphStatus reports graph build state.
func handleRepoGraphStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoPath := repoPathFromGraphRequest(r)
	if repoPath == "" {
		http.Error(w, "repo_path required", http.StatusBadRequest)
		return
	}
	rebuild := r.URL.Query().Get("rebuild") == "1" || r.URL.Query().Get("rebuild") == "true"
	if rebuild {
		graph.RebuildIndexAsync(repoPath)
	} else {
		meta, _ := graph.Status(repoPath)
		if !meta.Ready && !meta.Building {
			graph.BuildIndexAsync(repoPath)
		}
	}
	meta, err := graph.Status(repoPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}
