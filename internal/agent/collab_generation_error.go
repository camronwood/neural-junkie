package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// sendCollabVisibleGenerationError posts a collaboration_discussion line when an
// agent cannot complete a collab turn, so the panel and scenario harness see
// failure instead of silence.
func (a *Agent) sendCollabVisibleGenerationError(msg *protocol.Message, userMsg, errCode string, retryable bool) {
	if a == nil || msg == nil || a.Hub == nil {
		return
	}
	if a.effectiveChannelType(msg.Channel) != protocol.ChannelTypeCollaboration {
		return
	}

	collabID := msg.GetCollaborationID()
	phase := msg.GetCollaborationPhase()
	if collabID == "" {
		collabCtx := a.getCollaborationContext(msg)
		collabID = collabCtx.ID
		if phase == "" {
			phase = collabCtx.Phase
		}
	}
	if collabID == "" {
		return
	}

	body := fmt.Sprintf("**%s** could not complete this turn: %s", a.Info.Name, strings.TrimSpace(userMsg))
	collabMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		msg.Channel,
		a.Info,
		body,
	)
	collabMsg.ReplyTo = msg.ID
	collabMsg.SetCollaborationID(collabID)
	if phase != "" {
		collabMsg.SetCollaborationPhase(phase)
	}
	if msg.IsInThread() {
		collabMsg.ThreadID = msg.ThreadID
		collabMsg.IsThreadReply = true
	}
	if collabMsg.Metadata == nil {
		collabMsg.Metadata = make(map[string]interface{})
	}
	collabMsg.Metadata["generation_error"] = true
	if errCode != "" {
		collabMsg.SetErrorMetadata(errCode, retryable)
	}

	if sendErr := a.Hub.SendMessage(collabMsg); sendErr != nil {
		log.Printf("[%s] Failed to send collab-visible generation error: %v", a.Info.Name, sendErr)
		return
	}
	if a.Collab != nil {
		collabPhase := a.Collab.GetCollaboration(collabID, a.Info.ID).Phase
		if err := a.Collab.RecordMessage(collabID, collabMsg); err != nil {
			log.Printf("[%s] Warning: failed to record collab generation error: %v", a.Info.Name, err)
		}
		// Always hand off during planning so timeouts/errors do not strand silent participants.
		if collabPhase == "planning" && a.Collab.IsActive(collabID) {
			a.promptNextCollaborationTurn(collabMsg, collabID)
		}
	}
}

// sendGenerationFailureMessages surfaces errors to chat and, when in a collab channel,
// to collaboration_discussion so scenarios and the panel are not empty.
func (a *Agent) sendGenerationFailureMessages(msg *protocol.Message, err error) {
	userMsg, code, retryable := classifyUserFacingError(err)

	errMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		msg.Channel,
		a.Info,
		userMsg,
	)
	errMsg.ReplyTo = msg.ID
	errMsg.SetErrorMetadata(code, retryable)
	if msg.IsInThread() {
		errMsg.ThreadID = msg.ThreadID
		errMsg.IsThreadReply = true
	}
	if sendErr := a.Hub.SendMessage(errMsg); sendErr != nil {
		log.Printf("[%s] Failed to send error message: %v", a.Info.Name, sendErr)
	}

	a.sendCollabVisibleGenerationError(msg, userMsg, code, retryable)
}
