package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// resolveDMAgentForChannel returns the agent for dm-<user>-<agent> when channel.Agents is empty
// (e.g. after hub restart before the agent re-joins the DM).
func (ch *CommandHandler) resolveDMAgentForChannel(channel string) *protocol.AgentInfo {
	channel = strings.TrimSpace(channel)
	if !strings.HasPrefix(strings.ToLower(channel), "dm-") {
		return nil
	}
	parts := strings.SplitN(channel, "-", 3)
	if len(parts) != 3 {
		return nil
	}
	slug := strings.TrimSpace(parts[2])
	if slug == "" {
		return nil
	}
	// Channel slug is lowercase with no spaces (BiologyExpert → biologyexpert).
	if ag := ch.hub.FindLiveAgentByDisplayName(slug, ""); ag != nil {
		return ag
	}
	// Fallback: match agents whose slugified display name equals the channel slug.
	for _, info := range ch.hub.ListAgents() {
		if info == nil {
			continue
		}
		if slugifyAgentName(info.Name) == strings.ToLower(slug) {
			cp := *info
			return &cp
		}
	}
	return nil
}

func slugifyAgentName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

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
	if len(channelAgents) == 0 {
		if dmAgent := ch.resolveDMAgentForChannel(channel); dmAgent != nil {
			channelAgents = []protocol.AgentInfo{*dmAgent}
		}
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
		b.WriteString("• Enable **Life sciences** or **CAD** or **Software development** in Settings → Domain packs\n")
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
