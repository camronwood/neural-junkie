package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub/gitchange"
	"github.com/camronwood/neural-junkie/internal/orchestration"
)

const (
	durableInputUserQuestion = "user_question"
	durableInputToolApproval = "tool_approval"
	durableInputFileChange   = "file_change"
	durableInputGitChange    = "git_change"
	durableInputTaskApproval = "task_approval"
	durableInputPlanApproval = "plan_approval"
)

func (h *Hub) persistUserQuestion(q *UserQuestion, expiresAt time.Time) {
	if h == nil || q == nil {
		return
	}
	initial, err := json.Marshal(q)
	if err != nil {
		log.Printf("[Orchestration] marshal user question %s: %v", q.ID, err)
		return
	}
	schema := json.RawMessage(`{"type":"object","required":["answer"],"properties":{"answer":{"type":"string","minLength":1}}}`)
	h.persistDurableInput(orchestration.InputRequest{
		ID: q.ID, RunID: h.activeCollaborationID(q.Channel), Kind: durableInputUserQuestion,
		Schema: schema, InitialValue: initial, DecisionKey: q.DecisionKey,
		Requester: q.AgentID, ExpiresAt: expiresAt, CreatedAt: q.CreatedAt,
	})
}

func (h *Hub) persistToolApproval(a *ToolApproval, expiresAt time.Time) {
	if h == nil || a == nil {
		return
	}
	initial, err := json.Marshal(a)
	if err != nil {
		log.Printf("[Orchestration] marshal tool approval %s: %v", a.ID, err)
		return
	}
	schema := json.RawMessage(`{"type":"object","required":["status"],"properties":{"status":{"type":"string","enum":["approved","rejected"]},"reason":{"type":"string"}}}`)
	h.persistDurableInput(orchestration.InputRequest{
		ID: a.ID, RunID: h.activeCollaborationID(a.Channel), Kind: durableInputToolApproval,
		Schema: schema, InitialValue: initial, DecisionKey: a.SessionID + ":" + a.ToolName,
		Requester: a.AgentID, ExpiresAt: expiresAt, CreatedAt: a.CreatedAt,
	})
}

func (h *Hub) persistFileChange(change *filechange.FileChange) {
	if h == nil || change == nil {
		return
	}
	initial, err := json.Marshal(change)
	if err != nil {
		log.Printf("[Orchestration] marshal file change %s: %v", change.ID, err)
		return
	}
	h.persistDurableInput(orchestration.InputRequest{
		ID: change.ID, RunID: h.activeCollaborationID(change.Channel), Kind: durableInputFileChange,
		Schema:       json.RawMessage(`{"type":"object","required":["status"],"properties":{"status":{"type":"string","enum":["approved","rejected"]},"reason":{"type":"string"}}}`),
		InitialValue: initial, DecisionKey: "file:" + change.ID,
		Requester: change.Agent.ID, ExpiresAt: change.ExpiresAt, CreatedAt: change.RequestedAt,
	})
}

func (h *Hub) persistGitChange(proposal *gitchange.Proposal) {
	if h == nil || proposal == nil {
		return
	}
	initial, err := json.Marshal(proposal)
	if err != nil {
		log.Printf("[Orchestration] marshal git proposal %s: %v", proposal.ID, err)
		return
	}
	h.persistDurableInput(orchestration.InputRequest{
		ID: proposal.ID, RunID: h.activeCollaborationID(proposal.Channel), Kind: durableInputGitChange,
		Schema:       json.RawMessage(`{"type":"object","required":["status"],"properties":{"status":{"type":"string","enum":["approved","rejected","failed"]},"reason":{"type":"string"}}}`),
		InitialValue: initial, DecisionKey: "git:" + proposal.ID,
		Requester: proposal.Agent.ID, ExpiresAt: proposal.ExpiresAt, CreatedAt: proposal.RequestedAt,
	})
}

func (h *Hub) persistCollabTaskApproval(snapshot *collaboration.Collaboration, task collaboration.CollaborationTask) {
	if h == nil || snapshot == nil {
		return
	}
	initial, err := json.Marshal(task)
	if err != nil {
		return
	}
	h.persistDurableInput(orchestration.InputRequest{
		ID:    "task:" + snapshot.ID + ":" + task.ID,
		RunID: snapshot.ID, TaskID: task.ID, Kind: durableInputTaskApproval,
		Schema:       json.RawMessage(`{"type":"object","required":["status"],"properties":{"status":{"type":"string","enum":["approved","rejected"]},"reason":{"type":"string"}}}`),
		InitialValue: initial, DecisionKey: "task:" + task.ID,
		Requester: task.AssignedTo, CreatedAt: time.Now(),
	})
}

func (h *Hub) ResolveCollabTaskApproval(collabID, taskID, approver, status, reason string) {
	h.resolveDurableInput(
		"task:"+collabID+":"+taskID,
		approver,
		map[string]any{"status": status, "reason": reason},
	)
}

func (h *Hub) persistPlanApproval(snapshot *collaboration.Collaboration) {
	if h == nil || snapshot == nil || snapshot.ID == "" {
		return
	}
	initial, err := json.Marshal(snapshot.Plan)
	if err != nil {
		return
	}
	h.persistDurableInput(orchestration.InputRequest{
		ID: "plan:" + snapshot.ID, RunID: snapshot.ID, Kind: durableInputPlanApproval,
		Schema:       json.RawMessage(`{"type":"object","required":["status"],"properties":{"status":{"type":"string","enum":["approved","rejected"]},"reason":{"type":"string"}}}`),
		InitialValue: initial, DecisionKey: "plan_approval", Requester: snapshot.CreatedBy,
		CreatedAt: time.Now(),
	})
}

func (h *Hub) resolvePlanApproval(collabID, approver string) {
	h.resolveDurableInput("plan:"+collabID, approver, map[string]any{"status": "approved"})
}

func (h *Hub) ResolveGitProposalInput(id, approver, status, reason string) {
	h.resolveDurableInput(id, approver, map[string]any{"status": status, "reason": reason})
}

func (h *Hub) persistDurableInput(input orchestration.InputRequest) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	if input.RunID != "" {
		h.syncOrchestrationState(context.Background())
	}
	if _, err := runtime.store.CreateInput(context.Background(), input); err != nil &&
		!errors.Is(err, orchestration.ErrConflict) {
		log.Printf("[Orchestration] persist input %s: %v", input.ID, err)
	}
}

func (h *Hub) resolveDurableInput(id, approver string, response any) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	raw, err := json.Marshal(response)
	if err != nil {
		log.Printf("[Orchestration] marshal input response %s: %v", id, err)
		return
	}
	if _, err := runtime.store.ResolveInput(context.Background(), id, approver, raw); err != nil &&
		!errors.Is(err, orchestration.ErrAlreadyResolved) {
		log.Printf("[Orchestration] resolve input %s: %v", id, err)
	}
}

func (h *Hub) expireDurableInput(id, reason string) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	if err := runtime.store.ExpireInput(context.Background(), id, reason); err != nil &&
		!errors.Is(err, orchestration.ErrAlreadyResolved) {
		log.Printf("[Orchestration] expire input %s: %v", id, err)
	}
}

func (h *Hub) restoreDurableInputs() {
	if h == nil {
		return
	}
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	inputs, err := runtime.store.ListPendingInputs(context.Background())
	if err != nil {
		log.Printf("[Orchestration] restore durable inputs: %v", err)
		return
	}
	for _, input := range inputs {
		if !input.ExpiresAt.IsZero() && !input.ExpiresAt.After(time.Now()) {
			h.expireDurableInput(input.ID, "expired while hub was offline")
			continue
		}
		switch input.Kind {
		case durableInputUserQuestion:
			var q UserQuestion
			if err := json.Unmarshal(input.InitialValue, &q); err == nil {
				h.userQuestionManager.RestorePending(&q)
			}
		case durableInputToolApproval:
			var approval ToolApproval
			if err := json.Unmarshal(input.InitialValue, &approval); err == nil {
				h.toolApprovalManager.RestorePending(&approval)
			}
		case durableInputFileChange:
			var change filechange.FileChange
			if err := json.Unmarshal(input.InitialValue, &change); err == nil {
				h.fileChangeManager.RestorePending(&change)
				if root, _ := change.Metadata["workspace_root"].(string); root != "" {
					var workspaceIO filechange.WorkspaceIO
					if h.fileChangeBackendFn != nil {
						workspaceIO = h.fileChangeBackendFn(root)
					}
					_ = h.fileChangeManager.BindExecutionContext(change.ID, root, workspaceIO)
				}
			}
		case durableInputGitChange:
			var proposal gitchange.Proposal
			if err := json.Unmarshal(input.InitialValue, &proposal); err == nil {
				h.gitChangeManager.RestorePending(&proposal)
			}
		}
	}
}

// RestoreDurableOrchestrationInputs rehydrates unresolved user and tool gates.
func (h *Hub) RestoreDurableOrchestrationInputs() {
	h.restoreDurableInputs()
}

func (h *Hub) activeCollaborationID(channel string) string {
	if h == nil || h.collabManager == nil || channel == "" {
		return ""
	}
	for _, snapshot := range h.collabManager.ListActive() {
		if snapshot != nil && snapshot.Channel == channel {
			return snapshot.ID
		}
	}
	return ""
}
