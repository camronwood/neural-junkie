package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// GetDelegation exposes cross-agent consult for in-process agents.
func (h *Hub) GetDelegation() agent.DelegationClient {
	if h == nil || h.commandHandler == nil {
		return nil
	}
	return h.commandHandler
}

// DelegationEnabled implements agent.DelegationClient.
func (ch *CommandHandler) DelegationEnabled() bool {
	if ch == nil || ch.appConfig == nil {
		return false
	}
	return ch.appConfig.Delegation.Normalized().Enabled
}

// ResolveConsultants implements agent.DelegationClient.
func (ch *CommandHandler) ResolveConsultants(from protocol.AgentInfo, question string) []delegation.Candidate {
	if ch == nil || !ch.DelegationEnabled() {
		return nil
	}
	cfg := ch.appConfig.Delegation.Normalized()
	var candidates []protocol.AgentInfo
	for _, info := range ch.listRuntimeAgentInfos() {
		if info.ID == from.ID {
			continue
		}
		if skipDelegationTarget(info) {
			continue
		}
		candidates = append(candidates, info)
	}
	return delegation.Resolve(from, question, candidates, delegation.ResolveOptions{
		MinScore:       cfg.MinRelevanceScore,
		MaxCandidates:  cfg.MaxConsultsPerTurn,
		ExcludeAgentID: from.ID,
	})
}

// Consult implements agent.DelegationClient.
func (ch *CommandHandler) Consult(ctx context.Context, req delegation.ConsultRequest) (delegation.ConsultResult, error) {
	cfg := ch.appConfig.Delegation.Normalized()
	if !cfg.Enabled {
		return delegation.ConsultResult{}, fmt.Errorf("delegation disabled")
	}
	return ch.consultTarget(ctx, req, cfg)
}

// CollabVisibleConsult runs an in-process consult for collaboration L1 without requiring
// global chat delegation to be enabled. The hub posts the answer visibly in-channel.
func (ch *CommandHandler) CollabVisibleConsult(ctx context.Context, req delegation.ConsultRequest) (delegation.ConsultResult, error) {
	if ch == nil || ch.appConfig == nil {
		return delegation.ConsultResult{}, fmt.Errorf("command handler unavailable")
	}
	cfg := ch.appConfig.Delegation.Normalized()
	return ch.consultTarget(ctx, req, cfg)
}

func (ch *CommandHandler) consultTarget(ctx context.Context, req delegation.ConsultRequest, cfg config.DelegationConfig) (delegation.ConsultResult, error) {
	if req.FromID == req.ToID {
		return delegation.ConsultResult{}, fmt.Errorf("cannot consult self")
	}
	if req.Depth >= cfg.MaxDepth {
		return delegation.ConsultResult{}, fmt.Errorf("delegation max depth exceeded")
	}
	target, ok := ch.runtimeAgents[req.ToID]
	if !ok || target == nil {
		ch.agentsMu.RLock()
		repoAgent, repoOK := ch.repoAgents[req.ToID]
		ch.agentsMu.RUnlock()
		if repoOK && repoAgent != nil {
			text, err := repoAgent.GenerateConsultResponse(ctx, req.SubQuestion, req.Channel)
			if err != nil {
				return delegation.ConsultResult{AgentName: repoAgent.Info.Name, Err: err.Error()}, err
			}
			return delegation.ConsultResult{
				Text:      text,
				AgentName: repoAgent.Info.Name,
				Model:     repoAgent.GetAIProvider().GetModel(),
			}, nil
		}
		return delegation.ConsultResult{}, fmt.Errorf("consult target not in runtime: %s", req.ToID)
	}
	intent := req.Intent
	if intent == "" {
		intent = delegation.ClassifyForAgent(target.Info.Type, req.SubQuestion)
	}

	if intent == delegation.IntentDomainTools && len(target.AgentToolDefinitionsForConsult()) > 0 {
		text, err := target.GenerateConsultResponse(ctx, req.SubQuestion, intent, req.Channel)
		if err != nil {
			return delegation.ConsultResult{AgentName: target.Info.Name, Intent: intent, Err: err.Error()}, err
		}
		return delegation.ConsultResult{
			Text:      text,
			AgentName: target.Info.Name,
			Model:     target.GetAIProvider().GetModel(),
			Intent:    intent,
		}, nil
	}

	text, model, err := ch.modelConsult(ctx, target, req, cfg, intent)
	if err != nil {
		return delegation.ConsultResult{AgentName: target.Info.Name, Intent: intent, Err: err.Error()}, err
	}
	return delegation.ConsultResult{
		Text:      text,
		AgentName: target.Info.Name,
		Model:     model,
		Intent:    intent,
	}, nil
}

func (ch *CommandHandler) listRuntimeAgentInfos() []protocol.AgentInfo {
	if ch == nil {
		return nil
	}
	out := make([]protocol.AgentInfo, 0, len(ch.runtimeAgents))
	for _, a := range ch.runtimeAgents {
		if a != nil {
			out = append(out, a.Info)
		}
	}
	return out
}

func skipDelegationTarget(info protocol.AgentInfo) bool {
	return info.ConsultOnly
}

func (ch *CommandHandler) modelConsult(
	ctx context.Context,
	target *agent.Agent,
	req delegation.ConsultRequest,
	cfg config.DelegationConfig,
	intent delegation.Intent,
) (string, string, error) {
	if ch.providerCache == nil || ch.appConfig == nil {
		return "", "", fmt.Errorf("provider registry not configured")
	}
	acfg := ch.agentConfigForRuntime(target.Info.Name, target.Info.Type)
	if acfg == nil {
		return "", "", fmt.Errorf("no config for agent %q", target.Info.Name)
	}
	prov, err := ch.providerForConsult(*acfg, cfg, target.Info.Type, intent)
	if err != nil {
		return "", "", err
	}
	prompt := buildConsultPrompt(target.Info, req.SubQuestion)
	text, err := prov.GenerateResponse(ctx, prompt, nil)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(text), prov.GetModel(), nil
}

func (ch *CommandHandler) agentConfigForRuntime(name string, _ protocol.AgentType) *config.AgentConfig {
	for _, a := range ch.appConfig.EnabledAgents() {
		if a.Name == name {
			copy := a
			return &copy
		}
	}
	return nil
}

func (ch *CommandHandler) providerForConsult(
	acfg config.AgentConfig,
	cfg config.DelegationConfig,
	agentType protocol.AgentType,
	intent delegation.Intent,
) (ai.AIProvider, error) {
	p := ch.appConfig.ProviderForAgent(acfg)
	if p == nil {
		return nil, fmt.Errorf("no provider for agent %q", acfg.Name)
	}
	copy := *p
	if agentType == protocol.AgentTypeBiology || agentType == protocol.AgentTypeGenomics {
		if intent == delegation.IntentDomainReasoning {
			copy.Model = ch.appConfig.BiologyChatModelOrDefault()
		}
	}
	if ch.providerCache != nil {
		return ch.providerCache.GetForProviderRow(ch.appConfig, &copy)
	}
	return ai.ProviderFromConfig(&copy)
}

func buildConsultPrompt(info protocol.AgentInfo, subQuestion string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("You are %s, a %s specialist consulted internally by another Neural Junkie agent.\n", info.Name, info.Type))
	b.WriteString("Answer ONLY the sub-question below. Be concise and factual. Do not mention other agents or the consultation mechanism.\n")
	b.WriteString(ai.SystemPromptSeparator)
	b.WriteString(strings.TrimSpace(subQuestion))
	return b.String()
}
