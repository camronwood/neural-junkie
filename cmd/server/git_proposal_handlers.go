package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/git"
	"github.com/camronwood/neural-junkie/internal/hub/gitchange"
)

func handleGitChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		userID = "default"
	}
	pending := chatHub.GetGitChangeManager().ListPending(userID)
	if pending == nil {
		pending = []*gitchange.Proposal{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pending)
}

func handleGitChangeApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/git-changes/approve/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	mgr := chatHub.GetGitChangeManager()
	p, err := mgr.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if _, err := mgr.MarkApproved(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := executeGitProposal(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

func handleGitChangeReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/git-changes/reject/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if err := chatHub.GetGitChangeManager().Reject(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func executeGitProposal(p *gitchange.Proposal) error {
	if p == nil || strings.TrimSpace(p.WorkspaceID) == "" {
		return nil
	}
	ws, ok := workspaceManager.GetWorkspace(p.WorkspaceID)
	if !ok || ws == nil {
		return nil
	}
	root := ws.Path
	ctx := context.Background()
	switch p.Operation {
	case gitchange.OpStage:
		return git.Add(ctx, root, p.Paths)
	case gitchange.OpCommit:
		msg := p.Message
		if msg == "" {
			msg = "Neural Junkie agent commit"
		}
		return git.Commit(ctx, root, msg, p.Paths)
	case gitchange.OpPush:
		return git.Push(ctx, root)
	}
	return nil
}
