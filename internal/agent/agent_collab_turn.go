package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// PlanningHandoffHub is implemented by the chat hub for planning turn prompts.
// Optional on HubClient so unit-test stubs need not implement it.
type PlanningHandoffHub interface {
	DispatchPlanningHandoff(collabID, agentID string) bool
	KickPlanningDiscussionWatchdog(collabID string)
}

func (a *Agent) promptNextCollaborationTurn(source *protocol.Message, collabID string) {
	if source == nil || a.Collab == nil || !a.Collab.IsActive(collabID) {
		return
	}
	if a.Hub != nil && a.Hub.IsChannelHeld(source.Channel) {
		return
	}

	nextAgentID, err := a.Collab.GetCurrentTurnAgent(collabID)
	if err != nil || strings.TrimSpace(nextAgentID) == "" {
		return
	}
	if nextAgentID == a.Info.ID {
		// Turn stays on this agent after generation_error — re-prompt them to retry.
		if source == nil || source.From.ID != a.Info.ID || source.Metadata == nil {
			return
		}
		ge, ok := source.Metadata["generation_error"].(bool)
		if !ok || !ge {
			return
		}
	}

	collabInfo := a.Collab.GetCollaboration(collabID, a.Info.ID)
	if collabInfo.Phase == "reviewing" || collabInfo.Phase == "approved" || collabInfo.Phase == "executing" {
		return
	}

	a.requestPlanningTurnHandoff(collabID, nextAgentID)
}

func (a *Agent) requestPlanningTurnHandoff(collabID, nextAgentID string) bool {
	if a.Hub == nil {
		return false
	}
	if d, ok := a.Hub.(PlanningHandoffHub); ok {
		return d.DispatchPlanningHandoff(collabID, nextAgentID)
	}
	log.Printf("[%s] Warning: hub missing PlanningHandoffHub; skipping planning handoff", a.Info.Name)
	return false
}

func (a *Agent) kickPlanningTurnWatchdog(collabID string) {
	if a.Hub == nil || strings.TrimSpace(collabID) == "" {
		return
	}
	if d, ok := a.Hub.(PlanningHandoffHub); ok {
		d.KickPlanningDiscussionWatchdog(collabID)
	}
}

// SetMessageInterceptor sets an optional message pre-processing hook.
func collaborationWorkingDirectoryForMessage(a *Agent, msg *protocol.Message) string {
	if msg == nil || a == nil || a.Collab == nil {
		return ""
	}
	cid := msg.GetCollaborationID()
	if cid == "" {
		return ""
	}
	info := a.Collab.GetCollaboration(cid, a.Info.ID)
	if p := strings.TrimSpace(info.SourceRepoPath); p != "" {
		return p
	}
	if p := strings.TrimSpace(info.WorkingDirectory); p != "" {
		return p
	}
	return a.Collab.GetCollaborationWorkingDirectory(cid)
}

// syncWorkspaceFromMessage binds MCP tools and workspace scans to the collaboration
// source repo (or outbound workspace metadata) instead of a stale hub/editor root.
func (a *Agent) syncWorkspaceFromMessage(msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	if ws := collaborationWorkingDirectoryForMessage(a, msg); ws != "" {
		a.WorkspacePath = ws
		return
	}
	a.resolveWorkspacePath(msg)
}

// registerGenCancel tracks a cancellable generation for channel interject.
func isHumanCollabSpeaker(msg *protocol.Message) bool {
	if msg == nil || msg.IsFromSystem() {
		return false
	}
	return msg.From.Type == protocol.AgentTypeGeneral
}

// collabOutOfTurnMentionOK allows @mentions to wake an agent outside round-robin.
// During planning/review, only humans and system turn prompts may do so — not
// @mentions embedded in another agent's plan prose (which would skip participants).
func collabOutOfTurnMentionOK(msg *protocol.Message, phase string) bool {
	if msg == nil {
		return false
	}
	if msg.IsFromSystem() {
		return true
	}
	switch phase {
	case "planning", "reviewing":
		return isHumanCollabSpeaker(msg)
	default:
		return true
	}
}

func taskAssigneeFromMetadata(meta map[string]interface{}) (string, bool) {
	if meta == nil {
		return "", false
	}
	v, ok := meta["task_assigned_to"]
	if !ok || v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		return s, s != ""
	default:
		s := strings.TrimSpace(fmt.Sprint(x))
		return s, s != ""
	}
}

func recapAssigneeFromMetadata(meta map[string]interface{}) (string, bool) {
	if meta == nil {
		return "", false
	}
	v, ok := meta["recap_assignee"]
	if !ok || v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		return s, s != ""
	default:
		s := strings.TrimSpace(fmt.Sprint(x))
		return s, s != ""
	}
}

func (a *Agent) collabTaskRateLimitOK(collabID, taskID, dispatchToken string) bool {
	key := collabID
	if taskID != "" {
		key = collabID + ":" + taskID
	}
	a.collabTaskReplyMu.Lock()
	defer a.collabTaskReplyMu.Unlock()
	if a.collabTaskReplyAt == nil {
		a.collabTaskReplyAt = make(map[string]time.Time)
	}
	if a.collabTaskLastDispatchToken == nil {
		a.collabTaskLastDispatchToken = make(map[string]string)
	}
	token := strings.TrimSpace(dispatchToken)
	if token != "" && a.collabTaskLastDispatchToken[key] != token {
		// New dispatch token (approve/resume/watchdog redispatch) always wakes the assignee.
		a.collabTaskLastDispatchToken[key] = token
		a.collabTaskReplyAt[key] = time.Now()
		return true
	}
	if last, ok := a.collabTaskReplyAt[key]; ok && time.Since(last) < collabTaskMinReplyInterval {
		return false
	}
	a.collabTaskReplyAt[key] = time.Now()
	return true
}
