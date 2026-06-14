package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// AgentToolDefinitionsForConsult exposes MCP tools for hub delegation consults.
func (a *Agent) AgentToolDefinitionsForConsult() []ai.ClaudeToolDefinition {
	return a.agentToolDefinitions(nil)
}

// GenerateConsultResponse runs an internal consult without broadcasting to the channel.
func (a *Agent) GenerateConsultResponse(ctx context.Context, subQuestion string, intent delegation.Intent, channel string) (string, error) {
	subQuestion = strings.TrimSpace(subQuestion)
	if subQuestion == "" {
		return "", fmt.Errorf("empty consult question")
	}
	eff := a.GetAIProvider()
	if eff == nil {
		return "", fmt.Errorf("no AI provider")
	}
	prompt := buildInternalConsultPrompt(a.Info, subQuestion)
	msg := &protocol.Message{
		Channel: channel,
		Content: subQuestion,
		From: protocol.AgentInfo{
			ID:   "delegation",
			Name: "Delegation",
			Type: protocol.AgentTypeGeneral,
		},
	}
	if channel == "" {
		msg.Channel = "delegation-internal"
	}
	approvalCtx := ai.WithToolApprovalChannel(ctx, channel)
	if intent == delegation.IntentDomainTools && len(a.agentToolDefinitions(msg)) > 0 {
		return a.generateWithAgentTools(approvalCtx, msg, prompt, nil, eff)
	}
	return eff.GenerateResponse(approvalCtx, prompt, nil)
}

func buildInternalConsultPrompt(info protocol.AgentInfo, subQuestion string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("You are %s, a %s specialist consulted internally by another Neural Junkie agent.\n", info.Name, info.Type))
	b.WriteString("Answer ONLY the sub-question below. Be concise and factual.\n")
	b.WriteString(ai.SystemPromptSeparator)
	b.WriteString(subQuestion)
	return b.String()
}

func (a *Agent) shouldSkipDelegation(msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	if msg.GetCollaborationID() != "" {
		return true
	}
	switch msg.Type {
	case protocol.MessageTypeCollabTask, protocol.MessageTypeCollabRecap:
		return true
	}
	if msg.From.Type != "human" {
		return true
	}
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		return true
	}
	return false
}

func (a *Agent) appendDelegationContext(ctx context.Context, msg *protocol.Message, prompt string) string {
	dc := a.getDelegationClient()
	if dc == nil || !dc.DelegationEnabled() || a.shouldSkipDelegation(msg) {
		return prompt
	}
	if a.classifyTurnIntentForMessage(msg) == IntentClosure || a.classifyTurnIntentForMessage(msg) == IntentLowSignal {
		return prompt
	}
	candidates := dc.ResolveConsultants(a.Info, msg.Content)
	if len(candidates) == 0 {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\n=== DELEGATE_RESULTS ===\n")
	b.WriteString("The hub consulted other specialists for parts of this question. Merge their answers into your reply; do not invent facts they did not provide.\n")
	var consulted []string
	for _, c := range candidates {
		res, err := dc.Consult(ctx, delegation.ConsultRequest{
			FromID:      a.Info.ID,
			FromName:    a.Info.Name,
			ToID:        c.AgentID,
			SubQuestion: msg.Content,
			Channel:     msg.Channel,
			Depth:       0,
			Intent:      c.Intent,
		})
		if err != nil {
			log.Printf("[%s] delegation consult %s: %v", a.Info.Name, c.AgentName, err)
			continue
		}
		if strings.TrimSpace(res.Text) == "" {
			continue
		}
		consulted = append(consulted, res.AgentName)
		b.WriteString(fmt.Sprintf("\n--- %s (%s) ---\n%s\n", res.AgentName, res.Intent, strings.TrimSpace(res.Text)))
	}
	if len(consulted) == 0 {
		return prompt
	}
	a.setLastDelegationConsulted(consulted)
	b.WriteString("\n=== END DELEGATE_RESULTS ===\n")
	b.WriteString("Synthesize the above into one coherent answer for the user. Do not @mention other agents.\n")
	return b.String()
}

func (a *Agent) setLastDelegationConsulted(names []string) {
	a.delegationMu.Lock()
	a.lastDelegationConsulted = append([]string(nil), names...)
	a.delegationMu.Unlock()
}

// TakeDelegationConsulted returns consulted agent names for the last generation and clears the buffer.
func (a *Agent) TakeDelegationConsulted() []string {
	a.delegationMu.Lock()
	defer a.delegationMu.Unlock()
	out := a.lastDelegationConsulted
	a.lastDelegationConsulted = nil
	return out
}
