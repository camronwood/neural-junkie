package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/git"
	"github.com/camronwood/neural-junkie/internal/hub/gitchange"
	"github.com/camronwood/neural-junkie/internal/protocol"
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
	if _, err := mgr.MarkApplying(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	chatHub.UpdateChangeProposalStatus(
		p.Channel,
		p.ID,
		protocol.ChangeProposalStatusApplying,
		"",
		"",
	)
	if err := executeGitProposal(p); err != nil {
		_, _ = mgr.MarkFailed(id, err.Error())
		chatHub.ResolveGitProposalInput(id, "user", "failed", err.Error())
		chatHub.UpdateChangeProposalStatus(
			p.Channel,
			p.ID,
			protocol.ChangeProposalStatusFailed,
			"",
			err.Error(),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	approved, err := mgr.MarkApproved(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	chatHub.UpdateChangeProposalStatus(
		approved.Channel,
		approved.ID,
		protocol.ChangeProposalStatusApproved,
		"",
		"",
	)
	chatHub.ResolveGitProposalInput(id, "user", "approved", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approved)
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
	var req struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	proposal, err := chatHub.GetGitChangeManager().Reject(id, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	chatHub.UpdateChangeProposalStatus(
		proposal.Channel,
		proposal.ID,
		protocol.ChangeProposalStatusRejected,
		proposal.Reason,
		"",
	)
	chatHub.ResolveGitProposalInput(id, "user", "rejected", proposal.Reason)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(proposal)
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
