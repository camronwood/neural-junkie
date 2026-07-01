package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
)

const routingClassifierSystemPrompt = `You classify user tasks for an AI routing system.
Respond with JSON only (no markdown fences) using this schema:
{"domain":"general|security|biology|frontend|backend|devops|architecture|code_review|database|rust|cad","tool_need":true|false,"cost_tier":"cheap|standard|premium","confidence":0.0-1.0,"reason":"short_snake_case_reason","lora_tag":"optional nj-* tag when domain-specific adapter applies"}

Rules:
- security/auth/CVE/compliance tasks -> domain security, cost_tier premium
- typo/grammar/polish short tasks -> cost_tier cheap
- biology/protein/DNA/sequence -> domain biology
- react/css/frontend -> domain frontend
- api/sql/backend -> domain backend
- k8s/terraform/ci -> domain devops
- tool_need true when MCP tools, tests, linters, or file execution are clearly required`

// LLMClassifier uses a small utility model for structured routing decisions.
type LLMClassifier struct {
	Provider ai.AIProvider
	Timeout  time.Duration
}

// Classify calls the LLM and parses a RoutingDecision.
func (c LLMClassifier) Classify(ctx context.Context, in Input) (RoutingDecision, error) {
	if c.Provider == nil {
		return RoutingDecision{}, fmt.Errorf("routing llm classifier: nil provider")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prompt := routingClassifierSystemPrompt + "\n\n=== TASK ===\n"
	if in.AgentType != "" {
		prompt += "Agent type: " + in.AgentType + "\n"
	}
	prompt += in.Text

	raw, err := c.Provider.GenerateResponse(ctx, prompt, nil)
	if err != nil {
		return RoutingDecision{}, err
	}
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed struct {
		Domain     string  `json:"domain"`
		ToolNeed   bool    `json:"tool_need"`
		CostTier   string  `json:"cost_tier"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
		LoRATag    string  `json:"lora_tag"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return RoutingDecision{}, fmt.Errorf("parse routing json: %w (raw=%q)", err, truncate(raw, 200))
	}
	tag := strings.TrimSpace(parsed.LoRATag)
	if tag != "" && !tagInstalled(in.InstalledTags, tag) {
		tag = ""
	}
	return RoutingDecision{
		Domain:     parsed.Domain,
		ToolNeed:   parsed.ToolNeed,
		CostTier:   parsed.CostTier,
		Confidence: parsed.Confidence,
		Reason:     parsed.Reason,
		LoRATag:    tag,
		Source:     SourceLLM,
	}.Normalized(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// NewLLMClassifierFromProvider wraps an AI provider for routing classification.
func NewLLMClassifierFromProvider(p ai.AIProvider) LLMClassifier {
	return LLMClassifier{Provider: p, Timeout: 15 * time.Second}
}
