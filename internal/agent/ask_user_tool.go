package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/schema"
)

const askUserToolName = "ask_user"

// shouldOfferAskUserTool reports whether ask_user should be exposed for this turn.
// Symbol @codebase lookups and workspace-grounded guidance should answer from context.
func shouldOfferAskUserTool(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	if explicitCodebaseLookupWithChunks(msg) {
		return false
	}
	content := strings.TrimSpace(msg.Content)
	// Confused short follow-ups ("What?") need a plain chat clarification, not ask_user.
	if shortConfusedFollowUp(content) {
		return false
	}
	// Presence / vibe-check pings must be answered directly — never echoed as ask_user cards.
	if isConversationalOnlyTurn(msg) {
		return false
	}
	// Canonical semantic decision: only expose ask_user when the turn is actually ask_user.
	// Local models otherwise paraphrase presence checks into preference cards.
	if decision, ok := protocol.ExtractTurnDecision(msg); ok && decision.Action != intent.ActionAskUser {
		return false
	}
	if a == nil {
		return true
	}
	wsPath := strings.TrimSpace(a.resolveWorkspacePath(msg))
	if wsPath == "" && !messageHasWorkspaceContext(msg) {
		return true
	}
	if userRequestsImplementation(content) || userRequestsImplementationForMessage(a, msg) {
		return false
	}
	if workspaceGuidanceWithoutUserDecision(content) {
		return false
	}
	return true
}

// shortConfusedFollowUp reports ultra-short confusion / "say that again" turns.
// These must stay conversational so agents do not pivot into ask_user preference menus.
func shortConfusedFollowUp(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	// Strip a leading @mention so "@BackendEngineer What?" still matches.
	if strings.HasPrefix(lower, "@") {
		if i := strings.IndexAny(lower, " \t\n"); i >= 0 {
			lower = strings.TrimSpace(lower[i+1:])
		} else {
			return false
		}
	}
	switch lower {
	case "what?", "what", "huh?", "huh", "??", "???", "sorry?", "pardon?",
		"come again?", "come again", "what do you mean?", "what do you mean",
		"i don't understand", "i dont understand", "say that again?", "say that again":
		return true
	}
	if len(lower) <= 12 && strings.Count(lower, "?") >= 1 {
		trimmed := strings.Trim(lower, "?!. ")
		switch trimmed {
		case "what", "huh", "sorry", "pardon", "wait", "eh":
			return true
		}
	}
	return false
}

func explicitCodebaseLookupWithChunks(msg *protocol.Message) bool {
	if msg == nil || !codebaseMentionRE.MatchString(msg.Content) {
		return false
	}
	if msg.Metadata == nil {
		return false
	}
	switch v := msg.Metadata["injected_codebase_count"].(type) {
	case int:
		return v > 0
	case float64:
		return v > 0
	}
	return len(promptAttachmentsSlice(msg.Metadata[MetadataPromptAttachments])) > 0
}

func workspaceGuidanceWithoutUserDecision(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	needles := []string{
		"theme support", "theme toggle", "light and dark", "dark mode", "light mode",
		"add theme", "ui theme", "can you see my workspace", "see my workspace",
		"workspace i have open", "file tree",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

func appendAskUserToolPrompt(system *strings.Builder) {
	system.WriteString("USER QUESTIONS:\n")
	system.WriteString("When you need a decision, preference, or missing detail from the user, call the ask_user tool ONCE.\n")
	system.WriteString("Give the question a stable decision_key (for example \"target_platform\") so the same goal never asks it twice.\n")
	system.WriteString("After the user answers: continue the original task immediately. Do NOT ask the same (or nearly same) question again.\n")
	system.WriteString("Do NOT ask the user for directory listings or file contents you can obtain with list_dir / read_file / glob tools.\n")
	system.WriteString("If the user pasted an error log, diagnose and fix that error before asking new preference questions.\n\n")
}

func askUserToolDefinition() ai.ClaudeToolDefinition {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"question": {
				"type": "string",
				"description": "The question to ask the user. Be specific and concise."
			},
			"options": {
				"type": "array",
				"items": { "type": "string" },
				"description": "Optional multiple-choice options. Omit for free-text answers."
			},
			"goal_id": {
				"type": "string",
				"description": "Optional stable ID of the original user goal. Usually supplied by the host."
			},
			"decision_key": {
				"type": "string",
				"description": "Stable snake_case key for the decision within this goal, such as target_platform."
			}
		},
		"required": ["question"]
	}`)
	return ai.ClaudeToolDefinition{
		Name:        askUserToolName,
		Description: "Ask the user a clarifying question and wait for their answer. Use when you need a decision, preference, or missing detail — especially during collaborations.",
		InputSchema: schema,
	}
}

func (a *Agent) executeAskUserTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	args, err := schema.ParseInto[struct {
		Question    string   `json:"question"`
		Options     []string `json:"options"`
		GoalID      string   `json:"goal_id"`
		DecisionKey string   `json:"decision_key"`
	}](input, schema.ObjectSpec{Required: []string{"question"}})
	if err != nil {
		return "", fmt.Errorf("ask_user schema: %w", err)
	}
	args.Question = strings.TrimSpace(args.Question)
	if args.Question == "" {
		return "", fmt.Errorf("ask_user requires a non-empty question")
	}

	channel := msg.Channel
	turn := askUserTurnStateFromContext(ctx)
	if turn != nil && turn.count >= 1 {
		prior := strings.Join(turn.answered, "\n")
		if prior == "" {
			prior = "(prior answer recorded)"
		}
		return fmt.Sprintf(
			"ask_user was already used this turn. Prior answers:\n%s\n\nProceed with the user's original request now. Do not call ask_user again.",
			prior,
		), nil
	}

	// Keep this generation alive while blocked on the user (peer pause must not cancel us).
	a.pinGenerationForUserWait(channel)
	defer a.unpinGenerationForUserWait(channel)

	// Drop the "responding / using ask_user" indicator while we wait on the user.
	a.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)

	goalID := strings.TrimSpace(args.GoalID)
	if goalID == "" && msg.Metadata != nil {
		goalID = firstStringMetadata(msg.Metadata, "original_goal_id", "goal_id")
	}
	if resolver, ok := a.Hub.(interface {
		ResolveConversationGoalID(channel, explicitGoalID string) string
	}); ok {
		goalID = resolver.ResolveConversationGoalID(channel, goalID)
	}
	if goalID == "" {
		if goal, ok := turnGoalFromContext(ctx); ok {
			goalID = goal.ID
		}
		if goalID == "" {
			goalID = msg.ID
		}
	}
	var answer string
	contextualHub, supportsContext := a.Hub.(interface {
		AskUserQuestionWithContext(agentID, agentName, channel, question string, options []string, goalID, decisionKey string) (string, error)
	})
	if supportsContext {
		answer, err = contextualHub.AskUserQuestionWithContext(
			a.Info.ID, a.Info.Name, channel, args.Question, args.Options, goalID, strings.TrimSpace(args.DecisionKey),
		)
	} else {
		answer, err = a.Hub.AskUserQuestion(a.Info.ID, a.Info.Name, channel, args.Question, args.Options)
	}
	if err != nil {
		if ledger := actionEvidenceFromContext(ctx); ledger != nil {
			ledger.Record(ActionEvidence{Kind: EvidenceUserAnswer, Tool: askUserToolName, Status: "failed", Detail: err.Error()})
		}
		return fmt.Sprintf("User did not answer: %v", err), nil
	}
	if ledger := actionEvidenceFromContext(ctx); ledger != nil {
		ledger.Record(ActionEvidence{Kind: EvidenceUserAnswer, Tool: askUserToolName, Status: "succeeded"})
	}
	if turn != nil {
		turn.count++
		turn.answered = append(turn.answered, fmt.Sprintf("- %s → %s", args.Question, answer))
	}

	// Resume thinking for the remainder of the turn after the user answers.
	if ctx.Err() == nil {
		a.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)
	}
	return fmt.Sprintf(
		"User answered: %s\n\nContinue the original task now with this answer. Do not re-ask ask_user for the same information. Use workspace tools instead of asking for directory structure.",
		answer,
	), nil
}

func firstStringMetadata(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
