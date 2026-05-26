package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/scansummary"
)

const generateImageToolName = "generate_image"

var generateImageToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "prompt": {
      "type": "string",
      "description": "Detailed description of the image to generate"
    },
    "size": {
      "type": "string",
      "description": "Optional size (default 1024x1024)",
      "enum": ["1024x1024", "1792x1024", "1024x1792"]
    }
  },
  "required": ["prompt"]
}`)

func generateImageToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        generateImageToolName,
		Description: "Generate an image from a text prompt and post it to the current channel. Use when the user asks you to create, draw, or generate visual assets.",
		InputSchema: generateImageToolSchema,
	}
}

func (a *Agent) imageGenerationToolsEnabled() bool {
	return a.Info.SupportsImageGeneration && a.Hub != nil && a.Hub.ImageGenerationEnabled()
}

func (a *Agent) agentToolDefinitions() []ai.ClaudeToolDefinition {
	var tools []ai.ClaudeToolDefinition
	if a.imageGenerationToolsEnabled() {
		tools = append(tools, generateImageToolDefinition())
	}
	if a.MCPServer != nil {
		tools = append(tools, claudeToolsFromMCPServer(mcpServerFromInterface(a.MCPServer))...)
	}
	return tools
}

func appendImageGenerationPrompt(system *strings.Builder) {
	system.WriteString("IMAGE GENERATION:\n")
	system.WriteString("When the user asks you to create, draw, or generate an image, call the generate_image tool with a detailed prompt.\n")
	system.WriteString("After the tool succeeds, briefly confirm what you generated; the image is posted to the channel automatically.\n\n")
}

func (a *Agent) executeGenerateImageTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		Prompt string `json:"prompt"`
		Size   string `json:"size"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid generate_image input: %w", err)
		}
	}
	args.Prompt = strings.TrimSpace(args.Prompt)
	if args.Prompt == "" {
		return "", fmt.Errorf("generate_image requires a non-empty prompt")
	}
	if err := a.Hub.GenerateAndPostImage(ctx, msg.Channel, a.Info, args.Prompt, strings.TrimSpace(args.Size)); err != nil {
		return "", err
	}
	return "Image generated and posted to the channel.", nil
}

func (a *Agent) executeAgentTool(ctx context.Context, msg *protocol.Message, name string, input json.RawMessage) (string, error) {
	if name == generateImageToolName {
		return a.executeGenerateImageTool(ctx, msg, input)
	}
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return "", fmt.Errorf("tool %q not found", name)
	}
	input = rewriteScanSummaryToolInput(msg, name, input)
	return executeMCPTool(ctx, mcpServer, name, input)
}

func rewriteScanSummaryToolInput(msg *protocol.Message, name string, input json.RawMessage) json.RawMessage {
	if name != "summarize_scan_summary" {
		return input
	}
	sharedPath, ok := sharedScanSummaryPath(msg)
	if !ok || !scanSummaryPathExists(sharedPath) {
		return input
	}

	var args map[string]interface{}
	if len(input) > 0 {
		_ = json.Unmarshal(input, &args)
	}
	if args == nil {
		args = make(map[string]interface{})
	}
	current, _ := args["path"].(string)
	if strings.TrimSpace(current) != "" && scanSummaryPathExists(current) {
		return input
	}
	args["path"] = sharedPath
	out, err := json.Marshal(args)
	if err != nil {
		return input
	}
	return out
}

func sharedScanSummaryPath(msg *protocol.Message) (string, bool) {
	if msg == nil || msg.Metadata == nil {
		return "", false
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return "", false
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return "", false
	}
	workspacePath, _ := ctxMap["workspace_path"].(string)
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return "", false
	}

	summaryDir := ""
	if scan, ok := ctxMap["scan_summary"].(map[string]interface{}); ok {
		summaryDir, _ = scan["summary_dir"].(string)
	}
	summaryDir = strings.TrimSpace(summaryDir)
	if summaryDir == "" {
		return workspacePath, true
	}
	if filepath.IsAbs(summaryDir) {
		return summaryDir, true
	}
	return filepath.Join(workspacePath, summaryDir), true
}

func scanSummaryPathExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	dir, err := scansummary.ResolveSummaryDir(path)
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, scansummary.MetadataFileName)); err != nil {
		return false
	}
	return true
}

// generateWithAgentTools runs Claude tool-use for MCP and/or image generation tools.
func (a *Agent) generateWithAgentTools(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	history []*protocol.Message,
	eff ai.AIProvider,
) (string, error) {
	histMsgs := historyToMessages(history)
	tools := a.agentToolDefinitions()
	if len(tools) == 0 {
		return eff.GenerateResponse(ctx, prompt, histMsgs)
	}

	toolEff := a.toolCapableProvider(eff)
	toolProvider, ok := toolEff.(ai.ToolCapableProvider)
	if !ok || !toolProvider.SupportsTools() {
		log.Printf("[%s] Tools requested but provider does not support tool calling; using standard response", a.Info.Name)
		return eff.GenerateResponse(ctx, prompt, histMsgs)
	}

	return toolProvider.GenerateResponseWithTools(ctx, prompt, histMsgs, tools,
		func(ctx context.Context, req ai.ToolUseRequest) (string, error) {
			log.Printf("[%s] Tool call: %s", a.Info.Name, req.Name)
			return a.executeAgentTool(ctx, msg, req.Name, req.Input)
		})
}
