package hub

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

const (
	collabIdleRedispatchAfter = 90 * time.Second
	collabMaxTaskRedispatches = 2
)

// TickCollaborationIdleWatchdog heals post-approve stalls: workspace ack retry,
// pending dispatch, idle in-progress task redispatch, and silent planning discussions.
func (h *Hub) TickCollaborationIdleWatchdog(now time.Time) {
	if h == nil || h.collabManager == nil {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	for _, c := range h.collabManager.ListActive() {
		if c == nil {
			continue
		}
		if c.Phase == collaboration.PhasePlanning {
			h.collabManager.AdvancePlanningDiscussionIfTimedOut(c.ID)
			continue
		}
		if c.Phase != collaboration.PhaseExecuting {
			continue
		}
		h.tickCollaborationIdleWatchdogOne(c, now)
	}
}

func (h *Hub) tickCollaborationIdleWatchdogOne(c *collaboration.Collaboration, now time.Time) {
	if c == nil {
		return
	}
	channel := c.Channel
	if channel != "" && h.IsChannelHeld(channel) {
		return
	}
	if c.DispatchPaused {
		return
	}

	if !c.WorkspaceAcknowledged && collaboration.ShouldAutoAckWorkspaceOnApprove(c) {
		if h.collabWatchdogShouldRetryAutoAck(c.ID) {
			if err := h.AcknowledgeCollaborationWorkspace(c.ID, ""); err != nil {
				log.Printf("[Collaboration] Watchdog auto workspace ack for %s: %v", c.ID[:8], err)
				h.broadcastCollabSystem(channel, c.ID, fmt.Sprintf(
					"⚠️ **Workspace confirmation failed** (`%s`) — %v. Click **Continue** or run `/ack-collab-workspace %s`.",
					c.ID[:8], err, c.ID[:8],
				))
			}
		}
	}

	snap, err := h.collabManager.GetCollaborationSnapshot(c.ID)
	if err != nil || snap == nil {
		return
	}
	if !h.CollaborationCanDispatchTasks(snap) {
		return
	}

	hasReadyPending := false
	for _, task := range snap.Tasks {
		if task.Status == collaboration.TaskPending && !task.PromptDispatched && collaboration.IsTaskReadyForCollab(task, snap) {
			hasReadyPending = true
			break
		}
	}
	if hasReadyPending {
		if n := h.dispatchReadyCollabTasks(snap, nil, false); n > 0 {
			log.Printf("[Collaboration] Watchdog dispatched %d ready task(s) for %s", n, snap.ID[:8])
		}
		return
	}

	for _, task := range snap.Tasks {
		if task.Status != collaboration.TaskInProgress || !task.PromptDispatched {
			continue
		}
		if now.Sub(task.UpdatedAt) < collabIdleRedispatchAfter {
			continue
		}
		if h.isAssigneeBusy(task.AssignedTo) {
			continue
		}
		key := snap.ID + ":" + task.ID
		count := h.collabWatchdogRedispatchCount(key)
		if count >= collabMaxTaskRedispatches {
			if count == collabMaxTaskRedispatches {
				h.collabWatchdogBumpRedispatch(key)
				assignee := task.AssignedName
				if assignee == "" {
					assignee = "assignee"
				}
				h.broadcastCollabSystem(channel, snap.ID, fmt.Sprintf(
					"⚠️ **Task still idle** (`%s`) — @%s has not finished **%s**. Run `/resume-plan %s` to re-send prompts or mark done with `/collab-task-done`.",
					snap.ID[:8], assignee, task.Title, snap.ID[:8],
				))
			}
			continue
		}
		taskID := task.ID
		filter := func(t collaboration.CollaborationTask) bool { return t.ID == taskID }
		if n := h.dispatchCollabTaskMessagesFilter(snap, nil, filter, true); n > 0 {
			h.collabWatchdogBumpRedispatch(key)
			log.Printf("[Collaboration] Watchdog redispatched idle task %s for %s (attempt %d)", taskID[:8], snap.ID[:8], count+1)
		}
	}
}

func (h *Hub) isAssigneeBusy(agentID string) bool {
	if agentID == "" {
		return false
	}
	info, err := h.GetAgent(agentID)
	if err != nil || info == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(info.Status), "busy")
}

func (h *Hub) collabWatchdogShouldRetryAutoAck(collabID string) bool {
	h.collabWatchdogMu.Lock()
	defer h.collabWatchdogMu.Unlock()
	if h.collabWatchdogAutoAckTried == nil {
		h.collabWatchdogAutoAckTried = make(map[string]bool)
	}
	if h.collabWatchdogAutoAckTried[collabID] {
		return false
	}
	h.collabWatchdogAutoAckTried[collabID] = true
	return true
}

func (h *Hub) collabWatchdogRedispatchCount(key string) int {
	h.collabWatchdogMu.Lock()
	defer h.collabWatchdogMu.Unlock()
	if h.collabWatchdogRedispatch == nil {
		return 0
	}
	return h.collabWatchdogRedispatch[key]
}

func (h *Hub) collabWatchdogBumpRedispatch(key string) {
	h.collabWatchdogMu.Lock()
	defer h.collabWatchdogMu.Unlock()
	if h.collabWatchdogRedispatch == nil {
		h.collabWatchdogRedispatch = make(map[string]int)
	}
	h.collabWatchdogRedispatch[key]++
}
