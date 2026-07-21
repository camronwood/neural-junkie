package hub

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// GetDelegation exposes cross-agent consult for in-process agents.
func (h *Hub) GetDelegation() agent.DelegationClient {
	if h == nil || h.commandHandler == nil {
		return nil
	}
	return h.commandHandler
}

// CapabilityDirectory lists running providers for each executable capability.
func (ch *CommandHandler) CapabilityDirectory(fromAgentID string) []delegation.CapabilityProvider {
	if ch == nil || ch.appConfig == nil {
		return nil
	}
	var out []delegation.CapabilityProvider
	for _, cap := range ch.appConfig.ResolvedCapabilityRegistry().CapabilityRegistry {
		if !config.IsAgentAssignableCapability(cap) {
			continue
		}
		id := cap.QualifiedID
		if id == "" {
			id = cap.ID
		}
		row := delegation.CapabilityProvider{CapabilityID: id}
		for agentID, runtime := range ch.runtimeAgents {
			if runtime == nil || agentID == fromAgentID || !runtime.ProvidesCapability(id) {
				continue
			}
			row.AgentIDs = append(row.AgentIDs, runtime.Info.ID)
			row.AgentNames = append(row.AgentNames, runtime.Info.Name)
		}
		if len(row.AgentIDs) == 0 {
			continue
		}
		sort.Strings(row.AgentIDs)
		sort.Strings(row.AgentNames)
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CapabilityID < out[j].CapabilityID })
	return out
}

// RequestCapabilityHelp opens a visible, bounded handoff room and returns its result.
func (ch *CommandHandler) RequestCapabilityHelp(ctx context.Context, req delegation.CapabilityHelpRequest) (delegation.CapabilityHelpResult, error) {
	if ch == nil || ch.hub == nil || ch.appConfig == nil {
		return delegation.CapabilityHelpResult{}, fmt.Errorf("capability handoff unavailable")
	}
	if !ch.appConfig.CapabilityHandoffsEnabled() {
		return delegation.CapabilityHelpResult{}, fmt.Errorf("capability handoff is disabled")
	}
	if req.Depth >= 1 {
		return delegation.CapabilityHelpResult{}, fmt.Errorf("capability handoff depth exceeded")
	}
	req.Task = strings.TrimSpace(req.Task)
	req.CapabilityID = strings.TrimSpace(req.CapabilityID)
	if req.Task == "" || req.CapabilityID == "" || strings.TrimSpace(req.SourceChannel) == "" {
		return delegation.CapabilityHelpResult{}, fmt.Errorf("capability, bounded task, and source channel are required")
	}

	target, candidate, err := ch.resolveCapabilityHelper(req)
	if err != nil {
		return delegation.CapabilityHelpResult{}, err
	}
	handoffID := uuid.NewString()
	channelName := "handoff-" + handoffID[:8]
	room := ch.hub.CreateChannelWithType(
		channelName,
		fmt.Sprintf("Temporary %s handoff from %s to %s", req.CapabilityID, req.FromName, candidate.AgentName),
		"",
		protocol.ChannelTypeDelegation,
		req.CreatedBy,
	)
	ch.hub.mu.Lock()
	room.SourceChannel = req.SourceChannel
	room.SourceMessageID = req.SourceMessageID
	room.Tags = append(room.Tags, "capability-handoff", req.CapabilityID)
	ch.hub.mu.Unlock()
	if err := ch.hub.AddAgentToChannel(req.FromID, channelName); err != nil {
		return delegation.CapabilityHelpResult{}, err
	}
	if err := ch.hub.AddAgentToChannel(candidate.AgentID, channelName); err != nil {
		return delegation.CapabilityHelpResult{}, err
	}
	ch.hub.ensureAgentSubscribed(req.FromID, channelName)
	ch.hub.ensureAgentSubscribed(candidate.AgentID, channelName)

	now := time.Now()
	record := delegation.HandoffRecord{
		ID:              handoffID,
		SourceChannel:   req.SourceChannel,
		SourceMessageID: req.SourceMessageID,
		Channel:         channelName,
		RequestingID:    req.FromID,
		RequestingName:  req.FromName,
		HelperID:        candidate.AgentID,
		HelperName:      candidate.AgentName,
		CreatedBy:       req.CreatedBy,
		CapabilityID:    req.CapabilityID,
		Task:            req.Task,
		Status:          "running",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	ch.hub.upsertHandoff(record)
	ch.postHandoffEvent(req.SourceChannel, "handoff_started", record,
		fmt.Sprintf("Opening temporary capability handoff with @%s for `%s`.", candidate.AgentName, req.CapabilityID))
	ch.postHandoffEvent(channelName, "handoff_task", record,
		fmt.Sprintf("@%s\n\n%s", candidate.AgentName, req.Task))

	intent := delegation.ClassifyForAgent(target.Info.Type, req.Task)
	summary, consultErr := target.GenerateConsultResponse(ctx, req.Task, intent, channelName)
	record.UpdatedAt = time.Now()
	if consultErr != nil {
		record.Status = "failed"
		record.Error = consultErr.Error()
		ch.postHandoffEvent(channelName, "handoff_failed", record, fmt.Sprintf("Handoff failed: %v", consultErr))
		ch.postHandoffEvent(req.SourceChannel, "handoff_failed", record,
			fmt.Sprintf("Capability handoff to @%s failed: %v", candidate.AgentName, consultErr))
	} else {
		record.Status = "completed"
		record.Result = strings.TrimSpace(summary)
		ch.postHandoffEvent(channelName, "handoff_result", record,
			fmt.Sprintf("@%s completed the task:\n\n%s", candidate.AgentName, record.Result))
		ch.postHandoffEvent(req.SourceChannel, "handoff_completed", record,
			fmt.Sprintf("Capability handoff result from @%s:\n\n%s", candidate.AgentName, record.Result))
	}
	archiveTime := time.Now()
	record.ArchivedAt = &archiveTime
	_ = ch.hub.ArchiveChannel(channelName)
	target.RemoveChannel(channelName)
	if requester := ch.runtimeAgents[req.FromID]; requester != nil {
		requester.RemoveChannel(channelName)
	}
	ch.hub.upsertHandoff(record)

	result := delegation.CapabilityHelpResult{
		HandoffID:     handoffID,
		Channel:       channelName,
		SourceChannel: req.SourceChannel,
		HelperID:      candidate.AgentID,
		HelperName:    candidate.AgentName,
		Summary:       record.Result,
		Status:        record.Status,
	}
	if consultErr != nil {
		result.Err = consultErr.Error()
		return result, consultErr
	}
	return result, nil
}

func (ch *CommandHandler) resolveCapabilityHelper(req delegation.CapabilityHelpRequest) (*agent.Agent, delegation.Candidate, error) {
	var best *agent.Agent
	var candidate delegation.Candidate
	bestScore := -1
	for id, runtime := range ch.runtimeAgents {
		if runtime == nil || id == req.FromID || !runtime.ProvidesCapability(req.CapabilityID) {
			continue
		}
		score := delegation.RelevanceScore(runtime.Info, req.Task)
		if score > bestScore {
			best = runtime
			bestScore = score
			candidate = delegation.Candidate{
				AgentID:      runtime.Info.ID,
				AgentName:    runtime.Info.Name,
				AgentType:    runtime.Info.Type,
				Score:        score,
				Intent:       delegation.ClassifyForAgent(runtime.Info.Type, req.Task),
				CapabilityID: req.CapabilityID,
			}
		}
	}
	if best == nil {
		recommendation := "enable its pack or start the matching specialist"
		if cap, ok := packs.ResolveCapabilityQuery(ch.appConfig.ResolvedCapabilityRegistry().CapabilityRegistry, req.CapabilityID); ok && len(cap.MCPAgents) > 0 {
			recommendation = "start or ask one of: " + strings.Join(cap.MCPAgents, ", ")
		}
		return nil, delegation.Candidate{}, fmt.Errorf("no running agent can provide capability %q; %s", req.CapabilityID, recommendation)
	}
	return best, candidate, nil
}

func (ch *CommandHandler) postHandoffEvent(channel, kind string, record delegation.HandoffRecord, content string) {
	if ch == nil || ch.hub == nil {
		return
	}
	msg := &protocol.Message{
		ID:        uuid.NewString(),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   channel,
		From:      protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"handoff_event":    kind,
			"handoff_id":       record.ID,
			"handoff_channel":  record.Channel,
			"source_channel":   record.SourceChannel,
			"capability_id":    record.CapabilityID,
			"requesting_agent": record.RequestingID,
			"helper_agent":     record.HelperID,
		},
	}
	_ = ch.hub.SendMessage(msg)
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
