package hub

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CollabScheduler is the single owner of periodic and event-driven collaboration heals.
//
// Why a task/handoff was redispatched (state machine):
//
//	planning + silent turn holder
//	  -> OnPlanningSilence / Tick -> planning watchdog (15s throttle)
//	planning + need handoff (agent kick)
//	  -> OnPlanningNeedHandoff -> DispatchPlanningHandoff
//	executing + pending ready to run
//	  -> idle watchdog initial dispatch
//	executing + pending+PromptDispatched idle > 90s
//	  -> idle redispatch (max 2) then system warning
//	executing + generation_error on assignee reply
//	  -> OnGenerationError -> ClearTaskPromptDispatched + redispatch
//	planning cancel / RecordMessage failure
//	  -> OnPlanningKick (clears throttle then tick)
//
// Agent-side 15s handoff retries were removed so this scheduler is the sole timer owner.
type CollabScheduler struct {
	hub *Hub
}

// NewCollabScheduler wraps hub for naming clarity; Tick uses Hub methods.
func NewCollabScheduler(h *Hub) *CollabScheduler {
	if h == nil {
		return nil
	}
	return &CollabScheduler{hub: h}
}

// Tick runs planning silence + executing idle redispatches for all active collabs.
func (s *CollabScheduler) Tick(now time.Time) {
	if s == nil || s.hub == nil {
		return
	}
	s.OnPlanningSilence(now)
}

// OnPlanningSilence is the periodic entry for planning/executing idle watchdogs.
func (s *CollabScheduler) OnPlanningSilence(now time.Time) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.TickCollaborationIdleWatchdog(now)
}

// OnGenerationError clears prompt-dispatched and redispatches the failed task.
func (s *CollabScheduler) OnGenerationError(collabID string, msg *protocol.Message) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.maybeRedispatchAfterCollabGenerationError(msg, collabID)
}

// OnPlanningNeedHandoff sends a planning turn prompt for agentID.
func (s *CollabScheduler) OnPlanningNeedHandoff(collabID, agentID string) bool {
	if s == nil || s.hub == nil {
		return false
	}
	return s.hub.dispatchPlanningHandoff(collabID, agentID)
}

// OnPlanningKick clears handoff throttle and re-prompts silent planning participants.
func (s *CollabScheduler) OnPlanningKick(collabID string) {
	if s == nil || s.hub == nil {
		return
	}
	s.hub.kickPlanningDiscussionWatchdog(collabID)
}

// TickCollabScheduler is the preferred entry point from the server ticker.
func (h *Hub) TickCollabScheduler(now time.Time) {
	NewCollabScheduler(h).Tick(now)
}

// DispatchPlanningHandoff is the public hub API; heals go through CollabScheduler.
func (h *Hub) DispatchPlanningHandoff(collabID, agentID string) bool {
	return NewCollabScheduler(h).OnPlanningNeedHandoff(collabID, agentID)
}

// KickPlanningDiscussionWatchdog is the public hub API for planning silence heals.
func (h *Hub) KickPlanningDiscussionWatchdog(collabID string) {
	NewCollabScheduler(h).OnPlanningKick(collabID)
}
