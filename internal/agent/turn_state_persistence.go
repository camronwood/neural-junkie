package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type conversationGoalResolver interface {
	ResolveConversationGoalID(channel, explicitGoalID string) string
}

type conversationGoalPersister interface {
	PersistConversationGoal(channel, goalID, messageID, text string)
}

type conversationCorrectionRecorder interface {
	RecordConversationCorrection(channel, goalID, messageID, instruction string, supersedesMessageIDs []string)
}

type conversationActionPromiseRecorder interface {
	RecordConversationActionPromise(channel, actionID, goalID, action, description, messageID string)
}

type conversationActionCompleter interface {
	CompleteConversationAction(channel, actionID, messageID string) bool
}

// persistTurnConversationState bridges immutable agent TurnGoal data into the
// hub's durable channel state without adding hub types to the agent package.
func persistTurnConversationState(a *Agent, msg *protocol.Message, goal TurnGoal) TurnGoal {
	if msg == nil {
		return goal
	}
	stampTurnGoalCapabilities(msg, goal)
	if a == nil || a.Hub == nil {
		return goal
	}
	isCorrection := protocol.IsUserLikeSender(msg.From) && userCorrectionRE.MatchString(msg.Content)
	if goal.Action == ActionContinue || isCorrection ||
		(goal.Action == ActionArtifact && goal.ContinuationParent != "") {
		explicitGoalID := firstStringMetadata(msg.Metadata, "original_goal_id")
		if explicitGoalID == "" && goal.Action == ActionArtifact {
			explicitGoalID = strings.TrimSpace(goal.ContinuationParent)
		}
		retainedGoalID := ""
		if resolver, ok := a.Hub.(conversationGoalResolver); ok {
			retainedGoalID = resolver.ResolveConversationGoalID(msg.Channel, explicitGoalID)
		}
		if retainedGoalID == "" {
			retainedGoalID = explicitGoalID
		}
		if retainedGoalID == "" {
			retainedGoalID = firstStringMetadata(msg.Metadata, "goal_id")
		}
		if retainedGoalID != "" {
			goal.ID = retainedGoalID
		}
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["goal_id"] = goal.ID
	msg.Metadata["is_correction"] = isCorrection
	if goal.ID != msg.ID {
		msg.Metadata["original_goal_id"] = goal.ID
	}

	if persister, ok := a.Hub.(conversationGoalPersister); ok {
		persister.PersistConversationGoal(msg.Channel, goal.ID, msg.ID, goal.NormalizedRequest)
	}
	if isCorrection {
		if recorder, ok := a.Hub.(conversationCorrectionRecorder); ok {
			var supersedes []string
			if priorID := relevantPriorUserInstructionID(a.channelHistory(msg.Channel), msg); priorID != "" {
				supersedes = []string{priorID}
			}
			msg.Metadata["supersedes_message_ids"] = append([]string(nil), supersedes...)
			recorder.RecordConversationCorrection(
				msg.Channel, goal.ID, msg.ID, goal.NormalizedRequest, supersedes,
			)
		}
	}
	if goal.RequiresActionEvidence() {
		if recorder, ok := a.Hub.(conversationActionPromiseRecorder); ok {
			recorder.RecordConversationActionPromise(
				msg.Channel, conversationActionID(goal), goal.ID,
				string(goal.Action), goal.NormalizedRequest, msg.ID,
			)
		}
	}
	return goal
}

func stampTurnGoalCapabilities(msg *protocol.Message, goal TurnGoal) {
	if msg == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["turn_action"] = string(goal.Action)
	if !goal.ImplementationSession {
		return
	}
	msg.Metadata[protocol.IdeMetaImplementationSession] = true
	msg.Metadata[protocol.TurnMetaCanProposeFiles] = goal.Mutation == MutationWorkspace
	msg.Metadata[protocol.TurnMetaCanRunImplSession] = true
	msg.Metadata[protocol.TurnMetaRequiresWorkspace] = true
}

func relevantPriorUserInstructionID(history []*protocol.Message, current *protocol.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		prior := history[i]
		if prior == nil || prior.ID == "" || (current != nil && prior.ID == current.ID) ||
			!protocol.IsUserLikeSender(prior.From) || strings.TrimSpace(prior.Content) == "" {
			continue
		}
		// A bare approval is a continuation signal, not the instruction being corrected.
		if userAffirmsPendingImplementation(prior.Content) {
			continue
		}
		return prior.ID
	}
	return ""
}

func conversationActionID(goal TurnGoal) string {
	return strings.TrimSpace(goal.ID)
}

func completeValidatedConversationAction(a *Agent, msg *protocol.Message, goal TurnGoal, responseMessageID string) bool {
	if a == nil || a.Hub == nil || msg == nil || !goal.RequiresActionEvidence() {
		return false
	}
	completer, ok := a.Hub.(conversationActionCompleter)
	if !ok {
		return false
	}
	return completer.CompleteConversationAction(
		msg.Channel, conversationActionID(goal), responseMessageID,
	)
}

func (st *turnState) completePersistedAction() bool {
	if st == nil || !st.actionValidated || st.responseMsg == nil {
		return false
	}
	return completeValidatedConversationAction(
		st.agent, st.msg, st.goal, st.responseMsg.ID,
	)
}
