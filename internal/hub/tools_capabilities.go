package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// GetAgentToolCapabilities returns tool and model routing info for an agent.
func (ch *CommandHandler) GetAgentToolCapabilities(agentID string) (protocol.AgentToolCapabilities, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return protocol.AgentToolCapabilities{}, fmt.Errorf("agent_id required")
	}

	if ra := ch.resolveRuntimeAgent(agentID); ra != nil {
		if a, ok := ra.(*agent.Agent); ok && a != nil {
			return a.DescribeToolCapabilities(), nil
		}
	}

	info, err := ch.hub.GetAgent(agentID)
	if err != nil {
		return protocol.AgentToolCapabilities{}, err
	}
	return agent.CapabilitiesFromAgentInfo(info), nil
}

// ListChannelToolCapabilities returns tool info for agents in a channel (excludes moderator/system).
func (ch *CommandHandler) ListChannelToolCapabilities(channel string) (protocol.ChannelToolsResponse, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return protocol.ChannelToolsResponse{}, fmt.Errorf("channel required")
	}

	channelAgents, err := ch.hub.GetChannelAgents(channel)
	if err != nil {
		return protocol.ChannelToolsResponse{}, err
	}

	out := protocol.ChannelToolsResponse{Channel: channel}
	for _, ag := range channelAgents {
		if ag.Type == protocol.AgentTypeModerator || (ag.Type == protocol.AgentTypeGeneral && ag.Name == "System") {
			continue
		}
		cap, capErr := ch.GetAgentToolCapabilities(ag.ID)
		if capErr != nil {
			cap = agent.CapabilitiesFromAgentInfo(&ag)
			cap.Notes = append(cap.Notes, capErr.Error())
		}
		out.Agents = append(out.Agents, cap)
	}
	return out, nil
}

// ToolCountForAgent returns the number of hub tools for an agent (0 if unavailable).
func (ch *CommandHandler) ToolCountForAgent(agentID string) int {
	cap, err := ch.GetAgentToolCapabilities(agentID)
	if err != nil {
		return 0
	}
	return cap.ToolCount
}

func (ch *CommandHandler) handleToolsList(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	_ = ctx
	resp, err := ch.ListChannelToolCapabilities(msg.Channel)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	return ch.systemResponse(msg.Channel, formatChannelToolsList(resp)), nil
}

func formatChannelToolsList(resp protocol.ChannelToolsResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Tools in #%s**\n\n", resp.Channel)

	withTools := 0
	for _, ag := range resp.Agents {
		if ag.ToolCount > 0 {
			withTools++
		}
	}
	if withTools == 0 {
		b.WriteString("No agents in this channel currently expose hub MCP tools.\n\n")
		b.WriteString("• Enable **Life sciences** or **Software development** in Settings → Domain packs\n")
		b.WriteString("• Invite a specialist (e.g. `/create-expert biology` or add BiologyExpert to this channel)\n")
		b.WriteString("• Open agent **ℹ️** in the sidebar for per-agent tool details\n")
		return b.String()
	}

	for _, ag := range resp.Agents {
		if ag.ToolCount == 0 && len(ag.Notes) == 0 {
			continue
		}
		fmt.Fprintf(&b, "**%s** (%s)\n", ag.AgentName, ag.AgentType)
		if ag.ChatModel != "" {
			native := "native tools"
			if !ag.ChatNativeTools {
				native = "no native tools"
			}
			fmt.Fprintf(&b, "  Chat: %s / %s (%s)\n", ag.ChatProvider, ag.ChatModel, native)
		}
		if ag.ToolLoopModel != "" && (ag.ToolLoopUsesFallback || ag.ToolLoopModel != ag.ChatModel) {
			fmt.Fprintf(&b, "  Tool loop: %s\n", ag.ToolLoopModel)
		}
		if ag.MCPEnabled && ag.MCPPort > 0 {
			fmt.Fprintf(&b, "  MCP: localhost:%d\n", ag.MCPPort)
		}
		for _, tool := range ag.Tools {
			params := toolParamSummary(tool.Parameters)
			fmt.Fprintf(&b, "  • `%s`%s — %s\n", tool.Name, params, truncateToolDesc(tool.Description, 120))
		}
		for _, note := range ag.Notes {
			fmt.Fprintf(&b, "  _%s_\n", note)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func toolParamSummary(params []protocol.AgentToolParam) string {
	if len(params) == 0 {
		return ""
	}
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, p.Name)
	}
	return "(" + strings.Join(names, ", ") + ")"
}

func truncateToolDesc(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
