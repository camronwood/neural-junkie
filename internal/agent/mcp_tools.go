package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const maxToolResultChars = 12000

// mcpServerFromInterface returns the typed MCP server from MCPServerInterface.
func mcpServerFromInterface(m MCPServerInterface) *server.MCPServer {
	if m == nil {
		return nil
	}
	return m.GetMCPServer()
}

// claudeToolsFromMCPServer maps registered MCP tools to Anthropic tool definitions.
func claudeToolsFromMCPServer(mcpServer *server.MCPServer) []ai.ClaudeToolDefinition {
	if mcpServer == nil {
		return nil
	}
	tools := mcpServer.ListTools()
	out := make([]ai.ClaudeToolDefinition, 0, len(tools))
	for _, st := range tools {
		if st == nil {
			continue
		}
		schema, err := json.Marshal(st.Tool.InputSchema)
		if err != nil {
			continue
		}
		out = append(out, ai.ClaudeToolDefinition{
			Name:        st.Tool.Name,
			Description: st.Tool.Description,
			InputSchema: schema,
		})
	}
	return out
}

// executeMCPTool invokes a tool handler in-process on the agent's MCP server.
func executeMCPTool(ctx context.Context, mcpServer *server.MCPServer, name string, input json.RawMessage) (string, error) {
	st := mcpServer.GetTool(name)
	if st == nil {
		return "", fmt.Errorf("tool %q not found", name)
	}

	var args map[string]any
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid tool input: %w", err)
		}
	}

	req := mcpgo.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := st.Handler(ctx, req)
	if err != nil {
		return "", err
	}
	return formatCallToolResult(result), nil
}

func formatCallToolResult(result *mcpgo.CallToolResult) string {
	if result == nil {
		return ""
	}
	var parts []string
	for _, c := range result.Content {
		if tc, ok := c.(mcpgo.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	text := strings.Join(parts, "\n")
	if result.IsError {
		return "ERROR: " + text
	}
	if len(text) > maxToolResultChars {
		return text[:maxToolResultChars] + "\n...(truncated)"
	}
	return text
}

// appendMCPToolsPrompt adds dynamic tool descriptions to the system prompt.
func appendMCPToolsPrompt(system *strings.Builder, mcpServer *server.MCPServer, agentType protocol.AgentType) {
	if mcpServer == nil {
		return
	}
	tools := mcpServer.ListTools()
	if len(tools) == 0 {
		return
	}
	system.WriteString("AVAILABLE TOOLS:\n")
	switch agentType {
	case protocol.AgentTypeCAD:
		system.WriteString("You have access to the following OpenSCAD tools:\n")
	case protocol.AgentTypeBiology:
		system.WriteString("You have access to the following life-sciences analysis tools:\n")
	default:
		system.WriteString("You have access to the following diagnostic and analysis tools:\n")
	}
	for _, st := range tools {
		if st == nil {
			continue
		}
		system.WriteString(fmt.Sprintf("- %s: %s\n", st.Tool.Name, st.Tool.Description))
	}
	switch agentType {
	case protocol.AgentTypeCAD:
		system.WriteString("\nUse OpenSCAD tools when the user asks you to model, edit, render, or export geometry.\n")
		system.WriteString("When the user asks to create or write an .scad file, call write_openscad immediately — do not paste SCAD-only replies without calling the tool.\n")
		system.WriteString("Never print tool-call JSON or pseudo tool syntax in chat; use native tool calling only.\n")
		system.WriteString("For greetings, questions, and general chat, respond conversationally without calling tools.\n\n")
	case protocol.AgentTypeBiology:
		system.WriteString("\nUse biology tools when the user asks about sequences, structures, or scan data.\n")
		system.WriteString("When workspace context includes scan paths, call the matching summarize tool immediately.\n\n")
	default:
		system.WriteString("\nUse these tools to provide data-driven answers. When diagnosing issues,\n")
		system.WriteString("USE THE TOOLS to get actual data rather than guessing.\n\n")
	}
}

// generateWithMCPTools runs the AI provider tool loop when supported.
func (a *Agent) generateWithMCPTools(
	ctx context.Context,
	prompt string,
	history []*protocol.Message,
	eff ai.AIProvider,
) (string, error) {
	mcpServer := mcpServerFromInterface(a.MCPServer)
	histMsgs := historyToMessages(history)
	if mcpServer == nil {
		return eff.GenerateResponse(ctx, prompt, histMsgs)
	}

	toolProvider, ok := eff.(ai.ToolCapableProvider)
	if !ok || !toolProvider.SupportsTools() {
		log.Printf("[%s] MCP tools attached but provider does not support tool calling; using standard response", a.Info.Name)
		return eff.GenerateResponse(ctx, prompt, histMsgs)
	}

	tools := claudeToolsFromMCPServer(mcpServer)
	if len(tools) == 0 {
		return eff.GenerateResponse(ctx, prompt, histMsgs)
	}

	return toolProvider.GenerateResponseWithTools(withWebSearchGuard(ctx), prompt, histMsgs, tools,
		func(ctx context.Context, req ai.ToolUseRequest) (string, error) {
			log.Printf("[%s] MCP tool call: %s", a.Info.Name, req.Name)
			result, err := executeMCPTool(ctx, mcpServer, req.Name, req.Input)
			if err != nil {
				return result, err
			}
			return guardWebSearchToolResult(ctx, req.Name, result), nil
		})
}
