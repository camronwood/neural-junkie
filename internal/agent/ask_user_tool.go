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
