package hub

import (
	"context"
	"fmt"
	"log"

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
		SMSEnabled:   false,
	}
	if h != nil {
		h.collabActionConfigMu.RLock()
		cfg = h.collabActionConfig
		h.collabActionConfigMu.RUnlock()
	}
	return actions.NewRunner(cfg)
}

// executeCollabActionTask runs a hub action task and marks it complete on success.
func (h *Hub) executeCollabActionTask(snap *collaboration.Collaboration, task collaboration.CollaborationTask) bool {
	if h.collabManager == nil || snap == nil {
		return false
	}
	collabID := snap.ID
	runner := h.collabActionRunner()
	out, err := runner.Execute(context.Background(), snap, task)
	if err != nil {
		log.Printf("[Collaboration] action task %s failed: %v", task.ID[:8], err)
		_, _ = h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskBlocked, err.Error())
		h.broadcastCollabSystem(snap.Channel, collabID, fmt.Sprintf("🚫 Action **%s** failed: %v", task.Title, err))
		return false
	}
	effects, err := h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskCompleted, out)
	if err != nil {
		log.Printf("[Collaboration] action complete update %s: %v", task.ID[:8], err)
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
