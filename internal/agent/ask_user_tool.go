package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
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
	system.WriteString("When you need a decision, preference, or missing detail from the user, call the ask_user tool.\n")
	system.WriteString("Wait for the user's answer before proceeding — especially during collaborations.\n\n")
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
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}](input, schema.ObjectSpec{Required: []string{"question"}})
	if err != nil {
		return "", fmt.Errorf("ask_user schema: %w", err)
	}
	args.Question = strings.TrimSpace(args.Question)
	if args.Question == "" {
		return "", fmt.Errorf("ask_user requires a non-empty question")
	}

	channel := msg.Channel

	answer, err := a.Hub.AskUserQuestion(a.Info.ID, a.Info.Name, channel, args.Question, args.Options)
	if err != nil {
		return fmt.Sprintf("User did not answer: %v", err), nil
	}
	return fmt.Sprintf("User answered: %s", answer), nil
}
