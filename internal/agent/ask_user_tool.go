package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
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
	if a == nil {
		return true
	}
	wsPath := strings.TrimSpace(a.resolveWorkspacePath(msg))
	if wsPath == "" && !messageHasWorkspaceContext(msg) {
		return true
	}
	content := strings.TrimSpace(msg.Content)
	if userRequestsImplementation(content) || userRequestsImplementationForMessage(a, msg) {
		return false
	}
	if workspaceGuidanceWithoutUserDecision(content) {
		return false
	}
	return true
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
	var args struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid ask_user input: %w", err)
		}
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
