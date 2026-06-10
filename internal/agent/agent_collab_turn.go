package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (a *Agent) promptNextCollaborationTurn(source *protocol.Message, collabID string) {
	if source == nil || a.Collab == nil || !a.Collab.IsActive(collabID) {
		return
	}
	if a.Hub != nil && a.Hub.IsChannelHeld(source.Channel) {
		return
	}

	nextAgentID, err := a.Collab.GetCurrentTurnAgent(collabID)
	if err != nil || strings.TrimSpace(nextAgentID) == "" || nextAgentID == a.Info.ID {
		return
	}

	// Only prompt when the selected participant is currently eligible to respond.
	if !a.Collab.IsAgentTurn(collabID, nextAgentID) {
		return
	}

	collabInfo := a.Collab.GetCollaboration(collabID, a.Info.ID)
	if collabInfo.Phase == "reviewing" || collabInfo.Phase == "approved" || collabInfo.Phase == "executing" {
		return
	}

	handoffBody := collaborationTurnHandoffBody(collabInfo.Phase)
	if handoffBody == "" {
		return
	}

	turnMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		source.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		handoffBody,
	)
	turnMsg.SetCollaborationID(collabID)
	phase := collabInfo.Phase
	if phase == "" {
		phase = source.GetCollaborationPhase()
	}
	if phase != "" {
		turnMsg.SetCollaborationPhase(phase)
	}
	turnMsg.Mentions = []string{nextAgentID}
	if turnMsg.Metadata == nil {
		turnMsg.Metadata = map[string]interface{}{}
	}
	turnMsg.Metadata["collab_internal_event"] = true
	if messageHasWorkspaceContext(source) {
		if raw, ok := source.Metadata["workspace_context"]; ok {
			if ctx, ok := raw.(map[string]interface{}); ok {
				inheritWorkspaceContextFromCollaboration(turnMsg, ctx)
			}
		}
	} else {
		info := a.Collab.GetCollaboration(collabID, a.Info.ID)
		inheritWorkspaceContextFromCollaboration(turnMsg, info.SourceWorkspaceContext)
	}

	if err := a.Hub.SendMessage(turnMsg); err != nil {
		log.Printf("[%s] Warning: failed to send collaboration turn handoff: %v", a.Info.Name, err)
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

func (a *Agent) collabTaskRateLimitOK(collabID, taskID string) bool {
	key := collabID
	if taskID != "" {
		key = collabID + ":" + taskID
	}
	a.collabTaskReplyMu.Lock()
	defer a.collabTaskReplyMu.Unlock()
	if a.collabTaskReplyAt == nil {
		a.collabTaskReplyAt = make(map[string]time.Time)
	}
	if last, ok := a.collabTaskReplyAt[key]; ok && time.Since(last) < collabTaskMinReplyInterval {
		return false
	}
	a.collabTaskReplyAt[key] = time.Now()
	return true
}

// shouldRespond determines if the agent should respond to a message
