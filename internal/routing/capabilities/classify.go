package capabilities

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	unified "github.com/camronwood/neural-junkie/internal/routing"
)

// ClassifyCollabTask maps collab execution tasks to a capability task class.
func ClassifyCollabTask(taskText string) TaskClass {
	text := strings.TrimSpace(taskText)
	if text == "" {
		return TaskImplement
	}
	task := collaboration.CollaborationTask{Title: text, Description: text}
	if collaboration.TaskRequiresFileDeliverable(task) {
		return TaskImplement
	}
	if looksLightweightCollabTask(text) {
		return TaskCollabLight
	}
	return TaskImplement
}

func looksLightweightCollabTask(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	keywords := []string{
		"identify", "list ", "find relevant", "locate", "scan ", "explore",
		"inventory", "catalog", "enumerate", "discover", "grep", "search for",
		"analyze schema", "analyze the schema",
	}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return len(text) < 900
}

// ChatInput classifies normal chat/DM turns.
type ChatInput struct {
	Text        string
	AgentType   string
	AskMode     bool
	ImplSession bool
}

// ClassifyChat maps chat context to a capability task class.
func ClassifyChat(in ChatInput) TaskClass {
	if in.AskMode {
		return TaskAskMode
	}
	if in.ImplSession {
		return TaskImplement
	}
	text := strings.TrimSpace(in.Text)
	dec := unified.ClassifyRules(unified.Input{Text: text, AgentType: in.AgentType})
	if dec.CostTier == unified.CostCheap {
		return TaskUtility
	}
	return TaskChat
}

// ImplInput classifies implementation session routing.
type ImplInput struct {
	TaskText       string
	AgentType      string
	RepairAttempts int
	VerifyFailed   bool
	BootFixIntent  bool
}

// ClassifyImpl returns main and tool-loop task classes for implementation sessions.
func ClassifyImpl(in ImplInput) (main TaskClass, tool TaskClass) {
	// Reserve implement_heavy for repair/verify-failure tiers. Boot-fix and error keywords
	// on the first attempt stay on the standard implement class so local qwen2.5-coder /
	// qwen3.5 complete within live scenario timeouts (devstral:24b often exceeds 600–900s).
	if in.RepairAttempts >= 1 || in.VerifyFailed {
		return TaskImplementHeavy, TaskImplementHeavy
	}
	dec := unified.ClassifyRules(unified.Input{Text: in.TaskText, AgentType: in.AgentType})
	if dec.CostTier == unified.CostCheap {
		return TaskImplement, TaskUtility
	}
	return TaskImplement, TaskImplement
}
