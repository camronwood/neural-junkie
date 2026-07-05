package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	collabTurnHandoffRetryDelay = 25 * time.Second
	collabTurnHandoffMaxRetries = 3
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

	collabInfo := a.Collab.GetCollaboration(collabID, a.Info.ID)
	if collabInfo.Phase == "reviewing" || collabInfo.Phase == "approved" || collabInfo.Phase == "executing" {
		return
	}

	turnCount := a.Collab.ParticipantTurnCount(collabID, nextAgentID)
	if a.sendCollaborationTurnHandoff(source, collabID, nextAgentID, collabInfo.Phase) {
		a.scheduleCollaborationTurnHandoffRetry(source, collabID, nextAgentID, turnCount)
	}
}

func (a *Agent) sendCollaborationTurnHandoff(source *protocol.Message, collabID, nextAgentID, phase string) bool {
	if source == nil || a.Hub == nil || a.Collab == nil {
		return false
	}
	handoffBody := collaborationTurnHandoffBody(phase)
	if handoffBody == "" {
		return false
	}

	turnMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		source.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		handoffBody,
	)
	turnMsg.SetCollaborationID(collabID)
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
		return false
	}
	return true
}

func (a *Agent) scheduleCollaborationTurnHandoffRetry(source *protocol.Message, collabID, targetAgentID string, turnCountAtHandoff int) {
	if source == nil || a.Collab == nil {
		return
	}
	channel := source.Channel
	go func() {
		for attempt := 1; attempt <= collabTurnHandoffMaxRetries; attempt++ {
			time.Sleep(collabTurnHandoffRetryDelay)
			if a.Collab == nil || !a.Collab.IsActive(collabID) {
				return
			}
			if a.Hub != nil && a.Hub.IsChannelHeld(channel) {
				return
			}
			info := a.Collab.GetCollaboration(collabID, a.Info.ID)
			switch info.Phase {
			case "reviewing", "approved", "executing", "completed", "cancelled":
				return
			}
			if a.Collab.ParticipantTurnCount(collabID, targetAgentID) > turnCountAtHandoff {
				return
			}
			log.Printf("[%s] Collaboration turn handoff retry %d/%d for agent %s (collab %s)",
				a.Info.Name, attempt, collabTurnHandoffMaxRetries, targetAgentID, collabID[:8])
			retrySource := source
			if retrySource.Channel != channel {
				if work, err := protocol.CloneMessage(source); err == nil && work != nil {
					work.Channel = channel
					retrySource = work
				}
			}
			if !a.sendCollaborationTurnHandoff(retrySource, collabID, targetAgentID, info.Phase) {
				return
			}
		}
	}()
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
