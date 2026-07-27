package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/scananalysis"
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
	return a.imageGenerationToolsEnabledForMessage(nil)
}

func (a *Agent) imageGenerationToolsEnabledForMessage(msg *protocol.Message) bool {
	if a.Hub == nil || !a.Hub.ImageGenerationEnabled() {
		return false
	}
	if !agentTypeSupportsHubImageGen(a.Info.Type) {
		return false
	}
	if messageSuppressesImageGeneration(msg) {
		return false
	}
	return true
}

// messageSuppressesImageGeneration disables image tools during code/implementation work
// and collaboration planning/review turns so specialists stay on task lists, not image gen.
func messageSuppressesImageGeneration(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	// Neural Canvas deliverables never use FLUX — even if phrasing includes "diagram".
	if UserRequestsArtifact(msg.Content) || neuralCanvasDeliverableTurn(msg) {
		return true
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		if decision.Action == semantic.ActionArtifact {
			return true
		}
		if decision.Action == semantic.ActionImage {
			return false
		}
		// Workspace-mutation turns stay off image gen even if phrasing mentions "cover".
		if decision.Mutation == semantic.MutationWorkspace ||
			decision.Action == semantic.ActionEdit || decision.Action == semantic.ActionDebug ||
			decision.Action == semantic.ActionContinue || decision.Action == semantic.ActionRun {
			return true
		}
		// Answer/AskUser/etc.: honor an explicit generate-image or generate-music phrase.
		if UserRequestsGeneratedImage(msg.Content) || UserRequestsGeneratedMusic(msg.Content) {
			return false
		}
		return true
	}
	explicitImageIntent := UserRequestsGeneratedImage(msg.Content)
	if msg.ImplementationSession() {
		return true
	}
	// Active collaboration owns the turn even when ambient IDE metadata also
	// happens to describe a code editor.
	phase := strings.ToLower(strings.TrimSpace(msg.GetCollaborationPhase()))
	if phase != "" || msg.GetCollaborationID() != "" || msg.Type == protocol.MessageTypeCollabDiscussion {
		return true
	}
	// An explicit image request outranks passive IDE layout metadata. It does
	// not outrank an actual implementation session or collaboration phase.
	if explicitImageIntent {
		return false
	}
	if ConversationModeFromMessage(msg) == ConversationModeCode {
		return true
	}
	if msg.IdeEditorMode() == "agent" || msg.IdeEditorModeIsExport() {
		return true
	}
	// Shared by music tools: allow explicit song/track requests in chat.
	if UserRequestsGeneratedMusic(msg.Content) {
		return false
	}
	// Chat / unset mode: keep creative media tools off unless the user explicitly asked
	// (avoids "theme support" / workspace visibility false-positive tool calls).
	return true
}

func agentTypeSupportsHubImageGen(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeCLI, protocol.AgentTypeModerator, protocol.AgentTypeMaps:
		return false
	default:
		return true
	}
}

// tryHubImageGenerationShortcut posts a hub-generated image when the user asked for one.
func (a *Agent) tryHubImageGenerationShortcut(ctx context.Context, msg *protocol.Message) (string, bool) {
	explicitImageIntent := msg != nil && UserRequestsGeneratedImage(msg.Content)
	if goal, ok := turnGoalFromContext(ctx); ok && goal.Action == ActionImage {
		explicitImageIntent = true
	}
	// Map/route asks must never take the FLUX image path (even if turn goal says image).
	if msg != nil && UserRequestsMapOrRoute(msg.Content) {
		return "", false
	}
	// Neural Canvas / Mermaid asks must use create_artifact, not FLUX.
	if msg != nil && (UserRequestsArtifact(msg.Content) || neuralCanvasDeliverableTurn(msg)) {
		return "", false
	}
	if goal, ok := turnGoalFromContext(ctx); ok && goal.Action == ActionArtifact {
		return "", false
	}
	if a != nil && a.Info.Type == protocol.AgentTypeMaps {
		return "", false
	}
	if msg == nil || a.Hub == nil || protocol.IsGeneratedImageDelivery(msg) || !explicitImageIntent {
		return "", false
	}
	if !a.imageGenerationToolsEnabledForMessage(msg) {
		return "", false
	}
	prompt := ImagePromptFromMessage(msg.Content)
	if err := a.generateAndPostImageWithProgress(ctx, msg, StreamMessageIDFromContext(ctx), prompt, "", true); err != nil {
		return fmt.Sprintf("I couldn't generate that image: %v", err), true
	}
	return "Done — I've posted the generated image to the channel.", true
}

func (a *Agent) generateAndPostImageWithProgress(
	ctx context.Context,
	msg *protocol.Message,
	streamMsgID, prompt, size string,
	broadcastToolStart bool,
) error {
	a.sendThinkingActivity(msg, protocol.ThinkingActivityGeneratingImage, imageGenToolPreview(prompt))
	defer a.clearThinkingActivity(msg)

	if broadcastToolStart && streamMsgID != "" {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind:    "start",
			Name:    generateImageToolName,
			Preview: imageGenToolPreview(prompt),
		})
	}

	err := a.Hub.GenerateAndPostImage(ctx, msg.Channel, a.Info, prompt, size)
	if ledger := actionEvidenceFromContext(ctx); ledger != nil {
		status := "succeeded"
		if err != nil {
			status = "failed"
		}
		ledger.Record(ActionEvidence{
			Kind:   EvidenceImagePosted,
			Tool:   generateImageToolName,
			Status: status,
			Detail: prompt,
		})
	}

	if broadcastToolStart && streamMsgID != "" {
		ev := ai.ToolStepEvent{Kind: "done", Name: generateImageToolName, Preview: "Image ready"}
		if err != nil {
			ev.Kind = "error"
			ev.Preview = err.Error()
		}
		a.broadcastToolStep(ctx, msg, streamMsgID, ev)
	}
	return err
}

func imageGenToolPreview(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "Generating image…"
	}
	const max = 80
	if len(prompt) <= max {
		return "Generating image: " + prompt
	}
	return "Generating image: " + prompt[:max] + "…"
}

func (a *Agent) agentToolDefinitions(msg *protocol.Message) []ai.ClaudeToolDefinition {
	// Planning turns must produce a short prose/task-list response. Exposing
	// ask_user, workspace, or MCP tools lets local models enter a tool loop and
	// hold the round-robin turn until the scenario times out.
	if msg != nil && msg.Type == protocol.MessageTypeCollabDiscussion {
		switch msg.GetCollaborationPhase() {
		case "planning", "reviewing":
			return nil
		}
	}
	var tools []ai.ClaudeToolDefinition
	if a.imageGenerationToolsEnabledForMessage(msg) {
		tools = append(tools, generateImageToolDefinition())
	}
	if artifactToolsEnabledForMessage(msg) {
		tools = append(tools, artifactToolDefinitions()...)
	}
	if a.musicGenerationToolsEnabledForMessage(msg) {
		tools = append(tools, generateMusicToolDefinition(), extractStemsToolDefinition())
	}
	if a.mapsToolsEnabledForMessage(msg) {
		tools = append(tools, mapsCreateToolDefinition(), mapsUpdateToolDefinition())
	}
	if a.arenaToolsEnabledForMessage(msg) {
		tools = append(tools,
			arenaListChallengesToolDefinition(),
			arenaCreateSessionToolDefinition(),
			arenaGetStateToolDefinition(),
			arenaMakeMoveToolDefinition(),
			arenaSubmitAnswerToolDefinition(),
		)
	}
	if shouldOfferCapabilityTools(msg) {
		if activationTool, ok := a.activationToolDefinition(msg); ok {
			tools = append(tools, activationTool)
		}
		if helpTool, ok := a.capabilityHelpToolDefinition(msg); ok {
			tools = append(tools, helpTool)
		}
	}
	// Neural Canvas deliverables must not expose shell/file tools — local models
	// otherwise call `npx mermaid` / edit App.tsx instead of create_artifact.
	if neuralCanvasDeliverableTurn(msg) {
		if shouldOfferAskUserTool(a, msg) {
			tools = append(tools, askUserToolDefinition())
		}
		return tools
	}
	// Presence / casual answer turns must not receive workspace MCP tools.
	// Prompt tooling is already suppressed for IntentLowSignal, but the tool list
	// previously still exposed read_file/run_command and local models used them.
	if !isConversationalOnlyTurn(msg) {
		if a.MCPServer != nil {
			mcpTools := claudeToolsFromMCPServer(mcpServerFromInterface(a.MCPServer), effectiveMCPToolAllowlist(a, msg))
			tools = append(tools, a.filterToolsForActiveCapabilities(msg, mcpTools)...)
		}
		if a.hasWorkspaceTools() && !isAskModeReadOnly(msg) {
			tools = append(tools, fileEditToolDefinitions()...)
		}
	}
	if shouldOfferAskUserTool(a, msg) {
		tools = append(tools, askUserToolDefinition())
	}
	return tools
}

func appendImageGenerationPrompt(system *strings.Builder) {
	system.WriteString("IMAGE GENERATION:\n")
	system.WriteString("When the user asks you to create, draw, or generate an image, call the generate_image tool with a detailed prompt.\n")
	system.WriteString("After the tool succeeds, briefly confirm what you generated; the image is posted to the channel automatically.\n\n")
}

func (a *Agent) executeGenerateImageTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if messageSuppressesImageGeneration(msg) {
		return "", fmt.Errorf("generate_image is not available during implementation or code-editing sessions")
	}
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
	if err := a.generateAndPostImageWithProgress(ctx, msg, StreamMessageIDFromContext(ctx), args.Prompt, strings.TrimSpace(args.Size), false); err != nil {
		return "", err
	}
	return "Image generated and posted to the channel.", nil
}

func (a *Agent) executeAgentTool(ctx context.Context, msg *protocol.Message, name string, input json.RawMessage) (string, error) {
	if name == activateCapabilityToolName {
		return a.executeActivateCapabilityTool(msg, input)
	}
	if name == requestCapabilityHelpToolName {
		return a.executeRequestCapabilityHelpTool(ctx, msg, input)
	}
	if name == generateImageToolName {
		return a.executeGenerateImageTool(ctx, msg, input)
	}
	if name == createArtifactToolName || name == updateArtifactToolName {
		return a.executeArtifactTool(ctx, msg, name, input)
	}
	if name == generateMusicToolName {
		return a.executeGenerateMusicTool(ctx, msg, input)
	}
	if name == extractStemsToolName {
		return a.executeExtractStemsTool(ctx, msg, input)
	}
	if name == mapsCreateToolName {
		return a.executeMapsCreateTool(ctx, msg, input)
	}
	if name == mapsUpdateToolName {
		return a.executeMapsUpdateTool(ctx, msg, input)
	}
	if name == arenaCreateSessionToolName {
		return a.executeArenaCreateSessionTool(ctx, msg, input)
	}
	if name == arenaGetStateToolName {
		return a.executeArenaGetStateTool(ctx, msg, input)
	}
	if name == arenaMakeMoveToolName {
		return a.executeArenaMakeMoveTool(ctx, msg, input)
	}
	if name == arenaSubmitAnswerToolName {
		return a.executeArenaSubmitAnswerTool(ctx, msg, input)
	}
	if name == arenaListChallengesTool {
		return a.executeArenaListChallengesTool(ctx, msg, input)
	}
	if name == proposeFileEditToolName {
		return a.executeProposeFileEditTool(ctx, msg, input)
	}
	if name == searchReplaceToolName {
		return a.executeSearchReplaceTool(ctx, msg, input)
	}
	if name == applyPatchToolName {
		return a.executeApplyPatchTool(ctx, msg, input)
	}
	if name == askUserToolName {
		return a.executeAskUserTool(ctx, msg, input)
	}
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return "", fmt.Errorf("tool %q not found", name)
	}
	if collaborationDiscoveryToolBlocked(msg, name) {
		return "", fmt.Errorf("%s is unavailable for this focus-scoped collaboration task — read the provided source paths and ship the deliverable", name)
	}
	input = rewriteScanSummaryToolInput(msg, name, input)
	input = rewriteScanAnalysisToolInput(msg, name, input)
	input = a.rewriteCADToolInput(msg, name, input)
	wsRoot := a.resolveWorkspacePath(msg)
	if name == "read_file" || name == "get_file_content" {
		if path := parseReadFileToolInput(input); path != "" && !collaborationFocusReadPathAllowed(msg, wsRoot, path) {
			return collaborationFocusReadPathBlockMessage(msg, wsRoot, path), nil
		}
	}
	writtenPath := ""
	if name == "write_openscad" {
		writtenPath = cadWrittenPathFromToolInput(wsRoot, input)
	}
	toolCtx := a.contextWithWorkspaceBackend(ctx, msg)
	if wsRoot != "" {
		toolCtx = shared.ContextWithWorkspaceRoot(toolCtx, wsRoot)
	}
	if msg.ImplementationSession() || implementationSessionStateFromContext(ctx) != nil {
		toolCtx = shared.ContextWithImplementationSession(toolCtx, true)
		if st := implementationSessionStateFromContext(ctx); st != nil {
			toolCtx = attachImplSessionCommandPolicy(toolCtx, st)
		}
	}
	policy := commandPolicyFromContext(toolCtx)
	if name == "run_command" && policy != nil {
		cmd := parseRunCommandToolInput(input)
		if blockErr := policy.ShouldBlockRunCommand(cmd); blockErr != nil {
			return "ERROR: " + blockErr.Error(), nil
		}
	}
	if name == "run_command" {
		cmd := parseRunCommandToolInput(input)
		if cached, ok := lookupOrStoreRunCommandResult(ctx, cmd, "", false); ok {
			log.Printf("[%s] Skipping duplicate run_command this turn: %s", a.Info.Name, truncateImplLog(cmd, 120))
			return cached + "\n\n[Note: identical run_command already executed this turn; reused prior result.]", nil
		}
		toolCtx = shared.ContextWithRunCommandExtraAllows(toolCtx, mcpAppConfig().RunCommandAllowExtra())
		var allowErr error
		toolCtx, allowErr = a.maybeApproveRunCommand(toolCtx, msg, cmd)
		if allowErr != nil {
			errMsg := "ERROR: " + allowErr.Error()
			storeRunCommandTurnResult(ctx, cmd, errMsg)
			return errMsg, nil
		}
	}
	result, err := executeMCPTool(toolCtx, mcpServer, name, input)
	if name == "run_command" {
		cmd := parseRunCommandToolInput(input)
		if err == nil {
			storeRunCommandTurnResult(ctx, cmd, result)
		}
		if ledger := actionEvidenceFromContext(ctx); ledger != nil {
			code, _, _ := parseRunCommandMCPResult(result)
			status := "succeeded"
			if err != nil {
				status = "failed"
			}
			ledger.Record(ActionEvidence{Kind: EvidenceCommandRun, Tool: name, Status: status, ExitCode: &code})
			if err == nil && code == 0 {
				ledger.Record(ActionEvidence{Kind: EvidenceCommandPass, Tool: name, Status: "succeeded", ExitCode: &code})
			}
		}
		if policy != nil {
			exitCode, _, _ := parseRunCommandMCPResult(result)
			policy.RecordCommandRun(cmd, exitCode, result)
		}
		if err == nil {
			if !shouldSkipDuplicateCommandBroadcast(msg.Channel, a.Info.ID, cmd, result) {
				a.broadcastAgentRunCommandOutput(msg, cmd, result)
			}
		}
	}
	if name == "read_file" && err == nil && policy != nil {
		if path := parseReadFileToolInput(input); path != "" {
			policy.RecordReadPath(path)
		}
	}
	if err == nil && writtenPath != "" {
		a.trackCADFileWritten(wsRoot, writtenPath)
	}
	return result, err
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

	summaryDir := ""
	if scan, ok := ctxMap["scan_summary"].(map[string]interface{}); ok {
		summaryDir, _ = scan["summary_dir"].(string)
	}
	if strings.TrimSpace(summaryDir) == "" {
		if analysis, ok := ctxMap["scan_analysis"].(map[string]interface{}); ok {
			summaryDir, _ = analysis["linked_scan_dir"].(string)
		}
	}
	if strings.TrimSpace(summaryDir) == "" {
		if p, ok := scanDirFromActiveEditor(ctxMap, workspacePath, "scan_summary_dir"); ok {
			return p, true
		}
		if p, ok := scanDirFromOpenFiles(ctxMap, workspacePath, "scan_summary_dir"); ok {
			return p, true
		}
		if workspacePath != "" && scanSummaryPathExists(workspacePath) {
			return workspacePath, true
		}
		return "", false
	}
	return joinWorkspacePath(workspacePath, summaryDir), true
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

func rewriteScanAnalysisToolInput(msg *protocol.Message, name string, input json.RawMessage) json.RawMessage {
	if name != "summarize_scan_analysis" {
		return input
	}
	sharedPath, ok := sharedScanAnalysisPath(msg)
	if !ok || !scanAnalysisPathExists(sharedPath) {
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
	if strings.TrimSpace(current) != "" && scanAnalysisPathExists(current) {
		return input
	}
	args["path"] = sharedPath
	out, err := json.Marshal(args)
	if err != nil {
		return input
	}
	return out
}

func sharedScanAnalysisPath(msg *protocol.Message) (string, bool) {
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

	analysisDir := ""
	if scan, ok := ctxMap["scan_analysis"].(map[string]interface{}); ok {
		analysisDir, _ = scan["analysis_dir"].(string)
	}
	analysisDir = strings.TrimSpace(analysisDir)
	if analysisDir == "" {
		if p, ok := scanDirFromActiveEditor(ctxMap, workspacePath, "scan_analysis_dir"); ok {
			return p, true
		}
		if p, ok := scanPathFromActiveEditor(ctxMap, workspacePath); ok {
			return p, true
		}
		if p, ok := scanDirFromOpenFiles(ctxMap, workspacePath, "scan_analysis_dir"); ok {
			return p, true
		}
		if p, ok := scanPathFromOpenFiles(ctxMap, workspacePath); ok {
			return p, true
		}
		if workspacePath != "" && scanAnalysisPathExists(workspacePath) {
			return workspacePath, true
		}
		return "", false
	}
	return joinWorkspacePath(workspacePath, analysisDir), true
}

func joinWorkspacePath(workspacePath, dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return workspacePath
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	if workspacePath == "" {
		return dir
	}
	return filepath.Join(workspacePath, dir)
}

func scanDirFromActiveEditor(ctxMap map[string]interface{}, workspacePath, key string) (string, bool) {
	editor, ok := ctxMap["active_editor"].(map[string]interface{})
	if !ok {
		return "", false
	}
	dir, _ := editor[key].(string)
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", false
	}
	return joinWorkspacePath(workspacePath, dir), true
}

func scanPathFromActiveEditor(ctxMap map[string]interface{}, workspacePath string) (string, bool) {
	editor, ok := ctxMap["active_editor"].(map[string]interface{})
	if !ok {
		return "", false
	}
	path, _ := editor["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	if !filepath.IsAbs(path) && workspacePath != "" {
		path = filepath.Join(workspacePath, path)
	}
	if scanAnalysisPathExists(path) {
		return path, true
	}
	return "", false
}

func scanDirFromOpenFiles(ctxMap map[string]interface{}, workspacePath, key string) (string, bool) {
	files, ok := ctxMap["open_files"].([]interface{})
	if !ok {
		return "", false
	}
	for _, f := range files {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		isActive, _ := fm["is_active"].(bool)
		if !isActive {
			continue
		}
		dir, _ := fm[key].(string)
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		return joinWorkspacePath(workspacePath, dir), true
	}
	return "", false
}

func scanPathFromOpenFiles(ctxMap map[string]interface{}, workspacePath string) (string, bool) {
	files, ok := ctxMap["open_files"].([]interface{})
	if !ok {
		return "", false
	}
	for _, f := range files {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		isActive, _ := fm["is_active"].(bool)
		if !isActive {
			continue
		}
		path, _ := fm["path"].(string)
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) && workspacePath != "" {
			path = filepath.Join(workspacePath, path)
		}
		if scanAnalysisPathExists(path) {
			return path, true
		}
	}
	return "", false
}

func scanAnalysisPathExists(path string) bool {
	return scananalysis.IsAnalysisExport(path)
}

// generateWithAgentTools runs Claude tool-use for MCP and/or image generation tools.
func (a *Agent) generateWithAgentTools(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	history []*protocol.Message,
	eff ai.AIProvider,
) (string, error) {
	ctx = ai.EnsureToolLoopMaxIterations(ctx, chatToolLoopMaxIterations())
	histMsgs := historyToMessages(history)
	tools := a.agentToolDefinitions(msg)
	if len(tools) == 0 {
		return eff.GenerateResponse(ctx, prompt, histMsgs)
	}

	toolEff := a.toolCapableProvider(ctx, eff)
	a.broadcastRoutingTelemetry(msg)
	activeBefore := strings.Join(a.activeCapabilityIDs(msg), "\x00")
	text, err := a.runAgentToolLoop(ctx, msg, prompt, histMsgs, tools, toolEff, eff)
	if err != nil {
		return "", err
	}
	activeAfter := strings.Join(a.activeCapabilityIDs(msg), "\x00")
	if activeAfter != activeBefore {
		expanded := a.agentToolDefinitions(msg)
		if len(expanded) > len(tools) {
			text, err = a.runAgentToolLoop(
				ctx,
				msg,
				prompt+"\n\nA requested capability was activated for this turn. Continue the original task using its newly available tools.",
				histMsgs,
				expanded,
				toolEff,
				eff,
			)
			if err != nil {
				return "", err
			}
		}
	}
	return a.chainPlaintextToolResponse(ctx, msg, eff, prompt, histMsgs, text)
}

func (a *Agent) runAgentToolLoop(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	histMsgs []protocol.Message,
	tools []ai.ClaudeToolDefinition,
	toolEff ai.AIProvider,
	chatEff ai.AIProvider,
) (string, error) {
	toolProvider, ok := toolEff.(ai.ToolCapableProvider)
	if !ok || !toolProvider.SupportsTools() {
		log.Printf("[%s] Tools requested but provider does not support tool calling; using standard response", a.Info.Name)
		text, err := chatEff.GenerateResponse(ctx, prompt, histMsgs)
		if err != nil {
			return "", err
		}
		return text, nil
	}

	onToolUse := func(ctx context.Context, req ai.ToolUseRequest) (string, error) {
		log.Printf("[%s] Tool call: %s", a.Info.Name, req.Name)
		result, err := a.executeAgentTool(ctx, msg, req.Name, req.Input)
		if err != nil {
			return result, err
		}
		return guardWebSearchToolResult(ctx, req.Name, result), nil
	}

	toolCtx := withAskUserTurnState(withWebSearchGuard(ctx))
	toolCtx = withRunCommandTurnDedupe(toolCtx)
	text, err := toolProvider.GenerateResponseWithTools(toolCtx, prompt, histMsgs, tools, onToolUse)
	if errors.Is(err, ai.ErrNativeToolsUnsupported) {
		if react := wrapReActProvider(a, chatEff, chatEff.GetModel()); react != nil {
			a.broadcastRoutingTelemetry(msg)
			if tp, ok := react.(ai.ToolCapableProvider); ok {
				return tp.GenerateResponseWithTools(toolCtx, prompt, histMsgs, tools, onToolUse)
			}
		}
		if fb := a.ollamaToolSwapProvider(ctx, chatEff); fb != nil {
			if tp, ok := fb.(ai.ToolCapableProvider); ok && tp.SupportsTools() {
				log.Printf("[%s] Native tools failed (%v); using swap model", a.Info.Name, err)
				a.RecordRoutingSnapshot(RoutingSnapshot{
					Reason: "native_tools_fallback_swap",
					Source: "rules",
				})
				a.broadcastRoutingTelemetry(msg)
				return tp.GenerateResponseWithTools(toolCtx, prompt, histMsgs, tools, onToolUse)
			}
		}
		return "", err
	}
	if err != nil && ai.IsReActToolLoopError(err) {
		if fb := a.ollamaToolSwapProvider(ctx, chatEff); fb != nil {
			if tp, ok := fb.(ai.ToolCapableProvider); ok && tp.SupportsTools() {
				log.Printf("[%s] ReAct tool loop failed (%v); using swap model", a.Info.Name, err)
				a.RecordRoutingSnapshot(RoutingSnapshot{
					Reason: "react_fallback_swap",
					Source: "rules",
				})
				a.broadcastRoutingTelemetry(msg)
				return tp.GenerateResponseWithTools(toolCtx, prompt, histMsgs, tools, onToolUse)
			}
		}
		return "", err
	}
	return text, err
}

func implementationSessionActive(ctx context.Context) bool {
	if implementationSessionStateFromContext(ctx) != nil {
		return true
	}
	return shared.ImplementationSessionFromContext(ctx)
}

func isImplementationEditTool(name string) bool {
	switch strings.TrimSpace(name) {
	case proposeFileEditToolName, searchReplaceToolName, applyPatchToolName:
		return true
	default:
		return false
	}
}

// chainPlaintextToolResponse runs discover/diagnostic tools in a loop during implementation
// sessions instead of ending the round after the first plaintext JSON tool call.
func (a *Agent) chainPlaintextToolResponse(
	ctx context.Context,
	msg *protocol.Message,
	eff ai.AIProvider,
	prompt string,
	histMsgs []protocol.Message,
	text string,
) (string, error) {
	if !implementationSessionActive(ctx) {
		if recovered, ok := a.recoverPlaintextToolResponse(ctx, msg, text); ok {
			log.Printf("[%s] Recovered plaintext MCP tool call from chat model response", a.Info.Name)
			return recovered, nil
		}
		return text, nil
	}

	maxChain := ai.ToolLoopMaxIterationsFromContext(ctx)
	if maxChain <= 0 {
		maxChain = 8
	}
	lastText := strings.TrimSpace(text)
	for i := 0; i < maxChain; i++ {
		name, _, hasTool := ai.ParseToolCallFromText(lastText)
		if !hasTool {
			return lastText, nil
		}
		recovered, ok := a.recoverPlaintextToolResponse(ctx, msg, lastText)
		if !ok {
			return lastText, nil
		}
		log.Printf("[%s] Recovered plaintext MCP tool call from tool-loop model response (chain %d/%d)", a.Info.Name, i+1, maxChain)
		if isImplementationEditTool(name) {
			return recovered, nil
		}
		followUp := prompt + "\n\n=== TOOL RESULT (" + name + ") ===\n" +
			truncateImplLog(recovered, 4000) +
			"\n\nContinue the implementation session: use search_replace, apply_patch, or propose_file_edit to ship file changes. " +
			"Do not repeat read_file on paths you already read unless fixing a specific compile error.\n"
		next, err := eff.GenerateResponse(ctx, followUp, histMsgs)
		if err != nil {
			return recovered, nil
		}
		lastText = strings.TrimSpace(next)
	}
	log.Printf("[%s] Plaintext tool chain hit iteration cap", a.Info.Name)
	return lastText, nil
}

func (a *Agent) agentToolNames(msg *protocol.Message) map[string]bool {
	names := make(map[string]bool)
	for _, td := range a.agentToolDefinitions(msg) {
		names[td.Name] = true
	}
	return names
}

// recoverPlaintextToolResponse executes a tool when the model returned JSON in plain text.
func (a *Agent) recoverPlaintextToolResponse(ctx context.Context, msg *protocol.Message, text string) (string, bool) {
	name, input, ok := ai.ParseToolCallFromText(text)
	if !ok {
		return "", false
	}
	tools := a.agentToolNames(msg)
	if !tools[name] {
		return "", false
	}
	result, err := a.executeAgentTool(ctx, msg, name, input)
	if st := implementationSessionStateFromContext(ctx); st != nil && err == nil {
		st.recordDiscoverTool(name)
	}
	if err != nil {
		return fmt.Sprintf("Tool `%s` failed: %v", name, err), true
	}
	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("Tool `%s` completed with no output.", name), true
	}
	return result, true
}
