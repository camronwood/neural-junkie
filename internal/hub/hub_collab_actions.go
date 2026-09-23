package hub

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/collaboration/actions"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// SetCollabActionRunnerConfig configures hub-level action execution (HTTP allowlist, SMS, Slack, etc.).
func (h *Hub) SetCollabActionRunnerConfig(cfg actions.Config) {
	if h == nil {
		return
	}
	h.collabActionConfigMu.Lock()
	h.collabActionConfig = cfg
	h.collabActionConfigMu.Unlock()
}

func (h *Hub) collabActionRunner() *actions.Runner {
	cfg := actions.Config{
		AllowedHosts: nil,
	}
	if h != nil {
		h.collabActionConfigMu.RLock()
		cfg = h.collabActionConfig
		h.collabActionConfigMu.RUnlock()
	}
	return actions.NewRunner(cfg)
}

func isGatedNotifyAction(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "webhook", "sms", "email":
		return true
	default:
		return false
	}
}

// AfterCollabTaskApproved runs a gated notify action after user approval, or
// redispatches ready tasks for wait_human / other approvals.
func (h *Hub) AfterCollabTaskApproved(collabID, taskID string) {
	if h == nil || h.collabManager == nil {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return
	}
	var task *collaboration.CollaborationTask
	for i := range snap.Tasks {
		if snap.Tasks[i].ID == taskID {
			task = &snap.Tasks[i]
			break
		}
	}
	if task == nil {
		return
	}
	typ := ""
	if task.Action != nil {
		typ = strings.ToLower(strings.TrimSpace(task.Action.Type))
	}
	if isGatedNotifyAction(typ) {
		h.executeCollabActionTaskOpts(snap, *task, true)
		return
	}
	h.dispatchReadyCollabTasks(snap, nil, false)
}

// executeCollabActionTask runs a hub action task and marks it complete on success.
func (h *Hub) executeCollabActionTask(snap *collaboration.Collaboration, task collaboration.CollaborationTask) bool {
	return h.executeCollabActionTaskOpts(snap, task, false)
}

// executeCollabActionTaskOpts runs an action task. When approved is true, gated
// notify actions (webhook/sms/email) execute immediately without re-entering the approval gate.
func (h *Hub) executeCollabActionTaskOpts(snap *collaboration.Collaboration, task collaboration.CollaborationTask, approved bool) bool {
	if h.collabManager == nil || snap == nil {
		return false
	}
	if task.Status == collaboration.TaskCompleted {
		return false
	}
	collabID := snap.ID
	typ := ""
	if task.Action != nil {
		typ = strings.ToLower(strings.TrimSpace(task.Action.Type))
	}
	if typ == "wait_human" {
		_, _ = h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskInProgress, "Awaiting human approval")
		_ = h.collabManager.SetTaskAwaitingApproval(collabID, task.ID, true)
		h.persistCollabTaskApproval(snap, task)
		h.broadcastCollabSystem(snap.Channel, collabID, fmt.Sprintf("⏸ **%s** — waiting for your approval.", task.Title))
		return true
	}
	if task.AwaitingApproval && !approved {
		return false
	}
	if !approved && isGatedNotifyAction(typ) {
		if err := h.collabManager.SetTaskAwaitingApproval(collabID, task.ID, true); err == nil {
			h.persistCollabTaskApproval(snap, task)
			h.broadcastCollabSystem(snap.Channel, collabID, fmt.Sprintf("⏸ Action **%s** requires approval before running.", task.Title))
			return true
		}
	}
	runner := h.collabActionRunner()
	out, err := runner.Execute(context.Background(), snap, task)
	if err != nil {
		log.Printf("[Collaboration] action task %s failed: %v", shortCollabID(task.ID), err)
		_, _ = h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskBlocked, err.Error())
		h.broadcastCollabSystem(snap.Channel, collabID, fmt.Sprintf("🚫 Action **%s** failed: %v", task.Title, err))
		return false
	}
	effects, err := h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskCompleted, out)
	if err != nil {
		log.Printf("[Collaboration] action complete update %s: %v", shortCollabID(task.ID), err)
		return false
	}
	h.broadcastCollabSystem(snap.Channel, collabID, fmt.Sprintf("✅ Action **%s** completed.", task.Title))
	if effects.ShouldDispatchWave {
		if fresh, err := h.collabManager.GetCollaborationSnapshot(collabID); err == nil && fresh != nil {
			h.dispatchReadyCollabTasks(fresh, nil, false)
		}
	}
	return true
}

func (h *Hub) broadcastCollabSystem(channel, collabID, body string) {
	if channel == "" {
		return
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		body,
	)
	msg.SetCollaborationID(collabID)
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true
	_ = h.SendMessage(msg)
}
