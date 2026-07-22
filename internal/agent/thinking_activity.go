package agent

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func toolActivityDetail(ev ai.ToolStepEvent) string {
	name := strings.TrimSpace(ev.Name)
	if name == "" {
		name = "tool"
	}
	preview := strings.TrimSpace(ev.Preview)
	switch ev.Kind {
	case "start":
		if preview != "" {
			return name + " — " + preview
		}
		return name
	case "error":
		if preview != "" {
			return name + " failed — " + truncateThinkingDetail(preview)
		}
		return name + " failed"
	case "result", "done":
		if preview != "" {
			return name + " — " + truncateThinkingDetail(preview)
		}
		return name + " done"
	default:
		if preview != "" {
			return name + " — " + preview
		}
		return name
	}
}

func truncateThinkingDetail(s string) string {
	s = strings.TrimSpace(s)
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// sendThinkingActivity broadcasts a live activity label for the typing indicator.
func (a *Agent) sendThinkingActivity(originalMsg *protocol.Message, activity protocol.ThinkingActivity, detail ...string) {
	if originalMsg == nil || a.Hub == nil {
		return
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		originalMsg.Channel,
		a.Info,
		"",
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata["thinking_status"] = string(protocol.ThinkingStatusStarted)
	statusMsg.Metadata["question_id"] = originalMsg.ID
	if activity != "" {
		statusMsg.Metadata["thinking_activity"] = string(activity)
	}
	if len(detail) > 0 {
		if d := strings.TrimSpace(detail[0]); d != "" {
			statusMsg.Metadata[protocol.MetadataThinkingActivityDetail] = d
		}
	}
	go func() {
		if err := a.Hub.SendMessage(statusMsg); err != nil {
			log.Printf("[%s] Warning: failed to send thinking activity: %v", a.Info.Name, err)
		}
	}()
}

// clearThinkingActivity ends a scoped activity without resetting the footer to generic "is thinking".
func (a *Agent) clearThinkingActivity(originalMsg *protocol.Message) {
	_ = originalMsg
}

// broadcastRoutingTelemetry publishes live routing model info for the typing indicator and telemetry drawer.
func (a *Agent) broadcastRoutingTelemetry(originalMsg *protocol.Message) {
	if originalMsg == nil {
		return
	}
	snap := a.LastRoutingSnapshotFor(originalMsg.ID)
	detail := formatRoutingThinkingDetail(snap)
	if detail == "" {
		return
	}
	a.sendThinkingActivity(originalMsg, protocol.ThinkingActivityReasoning, detail)
	a.sendTelemetryEvent(originalMsg, "routing", routingTelemetryPayload(snap, originalMsg))
}

func routingTelemetryPayload(snap RoutingSnapshot, originalMsg *protocol.Message) map[string]interface{} {
	payload := map[string]interface{}{
		"chat_model":  snap.ChatModel,
		"tool_model":  snap.ToolModel,
		"provider_id": snap.ProviderID,
		"reason":      snap.Reason,
		"source":      snap.Source,
	}
	if snap.Domain != "" {
		payload["domain"] = snap.Domain
	}
	if snap.CostTier != "" {
		payload["cost_tier"] = snap.CostTier
	}
	if snap.KnowledgeRoute != "" {
		payload["knowledge_route"] = snap.KnowledgeRoute
	}
	if snap.KnowledgeReason != "" {
		payload["knowledge_reason"] = snap.KnowledgeReason
	}
	if len(snap.KnowledgeTargets) > 0 {
		payload["knowledge_targets"] = snap.KnowledgeTargets
	}
	if len(snap.KnowledgeExecuted) > 0 {
		payload["knowledge_executed"] = snap.KnowledgeExecuted
	}
	if snap.ConversationTier != "" {
		payload["conversation_tier"] = snap.ConversationTier
	}
	if len(snap.ConversationReasons) > 0 {
		payload["conversation_reasons"] = snap.ConversationReasons
	}
	if snap.ConversationEscalatedFrom != "" {
		payload["conversation_escalated_from"] = snap.ConversationEscalatedFrom
	}
	if gov := governanceTelemetryFromMessage(originalMsg); len(gov) > 0 {
		payload["governance"] = gov
	}
	return payload
}

func governanceTelemetryFromMessage(msg *protocol.Message) map[string]interface{} {
	if msg == nil {
		return nil
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	return map[string]interface{}{
		"composer_mode":        caps.ComposerMode,
		"context_scope":        caps.ContextTier,
		"can_propose_files":    caps.CanProposeFiles,
		"can_run_impl_session": caps.CanRunImplSession,
	}
}

func formatRoutingThinkingDetail(snap RoutingSnapshot) string {
	chat := strings.TrimSpace(snap.ChatModel)
	tool := strings.TrimSpace(snap.ToolModel)
	reason := strings.TrimSpace(snap.Reason)
	tier := strings.TrimSpace(snap.CostTier)
	route := strings.TrimSpace(snap.KnowledgeRoute)

	var parts []string
	if chat != "" {
		parts = append(parts, "chat: "+chat)
	}
	if tool != "" && tool != chat {
		parts = append(parts, "tools: "+tool)
	}
	var extras []string
	if tier != "" {
		extras = append(extras, "tier: "+tier)
	}
	if route != "" {
		extras = append(extras, "retrieval: "+route)
	}

	if len(parts) == 0 {
		if len(extras) == 0 {
			return ""
		}
		line := strings.Join(extras, " · ")
		if reason != "" {
			line += " (" + reason + ")"
		}
		return line
	}
	line := strings.Join(parts, " · ")
	if len(extras) > 0 {
		line += " · " + strings.Join(extras, " · ")
	}
	if reason != "" {
		line += " (" + reason + ")"
	}
	return line
}

func (a *Agent) sendTelemetryEvent(originalMsg *protocol.Message, kind string, payload map[string]interface{}) {
	if originalMsg == nil || a.Hub == nil || kind == "" {
		return
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		originalMsg.Channel,
		a.Info,
		"",
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata["thinking_status"] = string(protocol.ThinkingStatusStarted)
	statusMsg.Metadata["question_id"] = originalMsg.ID
	statusMsg.Metadata[protocol.MetadataTelemetryKind] = kind
	if len(payload) > 0 {
		statusMsg.Metadata[protocol.MetadataTelemetryPayload] = payload
	}
	go func() {
		if err := a.Hub.SendMessage(statusMsg); err != nil {
			log.Printf("[%s] Warning: failed to send telemetry event: %v", a.Info.Name, err)
		}
	}()
}

func (a *Agent) sendToolTelemetryEvent(originalMsg *protocol.Message, ev ai.ToolStepEvent) {
	if originalMsg == nil {
		return
	}
	payload := map[string]interface{}{
		"name": ev.Name,
		"kind": ev.Kind,
	}
	if ev.Preview != "" {
		payload["preview"] = ev.Preview
	}
	if ev.Iteration > 0 {
		payload["iteration"] = ev.Iteration
	}
	if ev.MaxIterations > 0 {
		payload["max_iterations"] = ev.MaxIterations
	}
	a.sendTelemetryEvent(originalMsg, "tool", payload)
}

// formatRoutingThinkingDetailForTest exposes routing detail formatting for tests.
func formatRoutingThinkingDetailForTest(snap RoutingSnapshot) string {
	return formatRoutingThinkingDetail(snap)
}

// routingTelemetryPayloadForTest exposes routing telemetry payload for tests.
func routingTelemetryPayloadForTest(snap RoutingSnapshot, msg *protocol.Message) map[string]interface{} {
	return routingTelemetryPayload(snap, msg)
}
