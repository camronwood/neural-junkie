package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	implSessionMaxToolIterations = 20
	implSessionMaxEditRounds     = 3
	implSessionTimeout           = 180 * time.Second
	editorTrustAutoApply         = "auto_apply_edits"
)

type implSessionStateKey struct{}
type implSessionRoundKey struct{}

// ImplementationSessionState tracks progress during a multi-step implementation run.
type ImplementationSessionState struct {
	EditRound       int
	FilesChanged    []string
	ProposedCount   int
	VerifyOutput    string
	VerifyFailed    bool
	VerifySkipped   bool
	RepairUsed      bool
	Phase           string
	StackManifest   *StackManifest
	SeedsLoaded     int
	DiscoverTools   []string
	PreflightErrors []string
	TrustMode       string
}

func withImplementationSessionState(ctx context.Context, s *ImplementationSessionState) context.Context {
	return context.WithValue(ctx, implSessionStateKey{}, s)
}

func implementationSessionStateFromContext(ctx context.Context) *ImplementationSessionState {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(implSessionStateKey{}).(*ImplementationSessionState)
	return s
}

func withImplementationSessionRound(ctx context.Context, round int) context.Context {
	return context.WithValue(ctx, implSessionRoundKey{}, round)
}

func implementationSessionRoundFromContext(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	r, _ := ctx.Value(implSessionRoundKey{}).(int)
	return r
}

// shouldRunImplementationSession reports whether to use the bounded implementation loop.
func shouldRunImplementationSession(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil {
		return false
	}
	if msg.GetCollaborationID() != "" {
		return false
	}
	if msg.IdeEditorMode() != "agent" {
		return false
	}
	if msg.IdeRouteAgentType() == "" && !msg.ImplementationSession() {
		return false
	}
	if !agentTypeCanShipFileChanges(a.Info.Type) {
		return false
	}
	return userRequestsImplementationForMessage(a, msg) || msg.ImplementationSession()
}

func (a *Agent) runImplementationSession(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, bool, []string, error) {
	text, _, proposed, files, err := a.runImplementationSessionStreaming(ctx, msg, eff, "")
	return text, proposed, files, err
}

func (a *Agent) runImplementationSessionStreaming(ctx context.Context, msg *protocol.Message, eff ai.AIProvider, streamMsgID string) (string, string, bool, []string, error) {
	sessionCtx, cancel := context.WithTimeout(ctx, implSessionTimeout)
	defer cancel()

	eff = a.EffectiveImplementationProvider(sessionCtx, msg)
	if eff == nil {
		eff = a.GetAIProvider()
	}

	toolModel := ""
	if globalImplementationRouting != nil {
		plan, _ := globalImplementationRouting.Plan(sessionCtx, eff, a.Info, msg)
		toolModel = plan.ToolModel
	}
	if toolModel == "" {
		toolModel = ai.ImplementationToolModelFromContext(sessionCtx)
	}
	if toolModel == "" {
		toolModel = "qwen2.5-coder:7b"
	}
	sessionCtx = ai.WithImplementationToolModel(sessionCtx, toolModel)
	sessionCtx = ai.WithToolLoopMaxIterations(sessionCtx, implSessionMaxToolIterations)

	state := &ImplementationSessionState{
		Phase:     "discover",
		TrustMode: msg.EditorAgentTrust(),
	}
	if wsPath := a.resolveWorkspacePath(msg); wsPath != "" {
		state.StackManifest = DetectStackManifest(wsPath)
	}
	sessionCtx = withImplementationSessionState(sessionCtx, state)

	var lastResponse string
	proposedAny := false
	var repairNote string

	for round := 0; round < implSessionMaxEditRounds; round++ {
		state.EditRound = round
		state.Phase = "discover"
		roundCtx := withImplementationSessionRound(sessionCtx, round)
		if repairNote != "" {
			roundCtx = withRepairNote(roundCtx, repairNote)
		}

		toolCtx := ai.WithToolStepObserver(roundCtx, func(ev ai.ToolStepEvent) {
			if ev.Kind == "result" && isDiscoverTool(ev.Name) {
				state.recordDiscoverTool(ev.Name)
			}
			if streamMsgID != "" {
				a.broadcastToolStep(roundCtx, msg, streamMsgID, ev)
			}
		})

		response, err := a.generateImplementationRound(toolCtx, msg, eff)
		if err != nil {
			return "", streamMsgID, proposedAny, state.FilesChanged, err
		}
		lastResponse = response

		proposalsBefore := state.ProposedCount
		cleaned, fileChangeProposed, propErr := a.maybeSubmitFileChangeFromResponse(toolCtx, response, msg.Channel, msg)
		toolProposed := state.ProposedCount > proposalsBefore

		if propErr != nil {
			log.Printf("[%s] impl session file proposal error: %v", a.Info.Name, propErr)
			repairNote = formatPreflightRepairNote(state.PreflightErrors, state.StackManifest)
			if round < implSessionMaxEditRounds-1 {
				continue
			}
		}

		if toolProposed || fileChangeProposed {
			proposedAny = true
			state.Phase = "edit"
			paths := extractChangedPathsFromResponse(response)
			state.FilesChanged = appendUnique(state.FilesChanged, paths)
			if toolProposed && len(paths) == 0 {
				// paths recorded in executeProposeFileEditTool via FilesChanged
			}
			lastResponse = cleaned
			break
		}

		if round < implSessionMaxEditRounds-1 {
			if len(state.PreflightErrors) > 0 {
				repairNote = formatPreflightRepairNote(state.PreflightErrors, state.StackManifest)
			} else if !state.groundingSatisfied() {
				repairNote = "Read the stack manifest and use read_file/glob_file_search on real paths before proposing edits."
			} else {
				repairNote = "You must emit [FILE_CHANGE] blocks or call propose_file_edit with real file paths and content. Advice-only replies do not satisfy this implementation request."
			}
		}
	}

	if proposedAny {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runImplementationVerify(sessionCtx, msg)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped

		if verifyFailed && !state.RepairUsed {
			state.RepairUsed = true
			state.Phase = "repair"
			repairNote = fmt.Sprintf("Verification failed. Fix the issues and emit corrected [FILE_CHANGE] blocks.\n\nCommand output:\n%s", verifyOut)
			roundCtx := withImplementationSessionRound(sessionCtx, state.EditRound+1)
			roundCtx = withRepairNote(roundCtx, repairNote)
			roundCtx = ai.WithToolLoopMaxIterations(roundCtx, implSessionMaxToolIterations)
			toolCtx := ai.WithToolStepObserver(roundCtx, func(ev ai.ToolStepEvent) {
				if ev.Kind == "result" && isDiscoverTool(ev.Name) {
					state.recordDiscoverTool(ev.Name)
				}
				if streamMsgID != "" {
					a.broadcastToolStep(roundCtx, msg, streamMsgID, ev)
				}
			})
			response, err := a.generateImplementationRound(toolCtx, msg, eff)
			if err == nil {
				proposalsBefore := state.ProposedCount
				cleaned, proposed, propErr := a.maybeSubmitFileChangeFromResponse(toolCtx, response, msg.Channel, msg)
				if propErr != nil {
					log.Printf("[%s] impl session repair proposal error: %v", a.Info.Name, propErr)
				}
				if proposed || state.ProposedCount > proposalsBefore {
					proposedAny = true
					lastResponse = cleaned
					verifyOut2, verifyFailed2, _ := a.runImplementationVerify(sessionCtx, msg)
					state.VerifyOutput = verifyOut2
					state.VerifyFailed = verifyFailed2
				} else {
					lastResponse = response
				}
			}
		}
	}

	summary := a.formatImplementationSessionSummary(lastResponse, state, proposedAny, msg)
	return summary, streamMsgID, proposedAny, state.FilesChanged, nil
}

type implRepairNoteKey struct{}

func withRepairNote(ctx context.Context, note string) context.Context {
	return context.WithValue(ctx, implRepairNoteKey{}, note)
}

func repairNoteFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	n, _ := ctx.Value(implRepairNoteKey{}).(string)
	return n
}

func (a *Agent) generateImplementationRound(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, error) {
	intent := a.classifyTurnIntentForMessage(msg)
	prompt := a.buildPromptForIntent(msg, intent)
	prompt = a.appendDelegationContext(ctx, msg, prompt)

	if note := repairNoteFromContext(ctx); note != "" {
		prompt += "\n=== REPAIR REQUIRED ===\n" + note + "\n"
	}

	if round := implementationSessionRoundFromContext(ctx); round == 0 {
		wsPath := a.resolveWorkspacePath(msg)
		if wsPath != "" && a.shouldAugmentPromptWithWorkspace(intent, msg) {
			if st := implementationSessionStateFromContext(ctx); st != nil && st.StackManifest != nil {
				prompt += st.StackManifest.FormatPromptBlock()
			}
			var referencedFiles strings.Builder
			AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
			seeds := AppendImplementationSeedFiles(&referencedFiles, a, msg, wsPath, a.Info.Type, collectIncludedFilePaths(msg))
			if st := implementationSessionStateFromContext(ctx); st != nil {
				st.SeedsLoaded += seeds
			}
			prompt += referencedFiles.String()
		}
	} else {
		prompt += "\n=== IMPLEMENTATION SESSION ===\nUse grep, glob_file_search, semantic_search, and read_file to find code. Do not guess file contents.\n\n"
	}

	history := a.conversationHistoryForIntent(msg, intent)
	prompt, _ = applyContextBudgetForMessage(msg, prompt)

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	if len(a.agentToolDefinitions()) > 0 {
		return a.generateWithAgentTools(approvalCtx, msg, prompt, history, eff)
	}
	return eff.GenerateResponse(approvalCtx, prompt, historyToMessages(history))
}

func (a *Agent) runImplementationVerify(ctx context.Context, msg *protocol.Message) (output string, failed bool, skipped bool) {
	if msg != nil && msg.EditorAgentTrust() != editorTrustAutoApply {
		return "", false, true
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return "", false, true
	}
	cmds := detectVerifyCommands(wsPath)
	if len(cmds) == 0 {
		return "", false, true
	}
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return "", false, true
	}

	var combined strings.Builder
	anyFailed := false
	for _, cmd := range cmds {
		input, _ := json.Marshal(map[string]string{"command": cmd})
		result, err := executeMCPTool(ctx, mcpServer, "run_command", input)
		if combined.Len() > 0 {
			combined.WriteString("\n---\n")
		}
		combined.WriteString("$ ")
		combined.WriteString(cmd)
		combined.WriteString("\n")
		if err != nil {
			combined.WriteString(err.Error())
			anyFailed = true
			break
		}
		combined.WriteString(result)
		if strings.Contains(result, "exit_code=") && !strings.Contains(result, "exit_code=0") {
			anyFailed = true
			break
		}
	}
	return combined.String(), anyFailed, false
}

func detectVerifyCommands(wsPath string) []string {
	if _, err := os.Stat(filepath.Join(wsPath, "go.mod")); err == nil {
		return []string{"go test ./..."}
	}
	if _, err := os.Stat(filepath.Join(wsPath, "Cargo.toml")); err == nil {
		return []string{"cargo test"}
	}
	if _, err := os.Stat(filepath.Join(wsPath, "package.json")); err == nil {
		return detectNodeVerifyCommands(wsPath)
	}
	return nil
}

func detectNodeVerifyCommands(wsPath string) []string {
	var cmds []string
	if hasPackageScript(wsPath, "build") {
		cmds = append(cmds, "npm run build")
	} else if _, err := os.Stat(filepath.Join(wsPath, "tsconfig.json")); err == nil {
		cmds = append(cmds, "npx tsc --noEmit")
	}
	if hasPackageScript(wsPath, "typecheck") {
		cmds = append(cmds, "npm run typecheck")
	}
	cmds = append(cmds, "npm test --if-present")
	return cmds
}

func hasPackageScript(wsPath, name string) bool {
	b, err := os.ReadFile(filepath.Join(wsPath, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	_, ok := pkg.Scripts[name]
	return ok
}

func (a *Agent) formatImplementationSessionSummary(lastResponse string, state *ImplementationSessionState, proposed bool, msg *protocol.Message) string {
	var b strings.Builder
	trust := ""
	if msg != nil {
		trust = msg.EditorAgentTrust()
	}
	if state != nil && state.TrustMode != "" {
		trust = state.TrustMode
	}

	switch {
	case !proposed:
		b.WriteString("Implementation session finished without file changes.\n\n")
	case trust == editorTrustAutoApply && state != nil && state.VerifyFailed:
		if len(state.FilesChanged) > 0 {
			b.WriteString(fmt.Sprintf("Implementation session complete — applied but verification failed (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
		} else {
			b.WriteString("Implementation session complete — applied but verification failed.\n\n")
		}
	case trust == editorTrustAutoApply && state != nil && !state.VerifyFailed && !state.VerifySkipped:
		if len(state.FilesChanged) > 0 {
			b.WriteString(fmt.Sprintf("Implementation session complete — applied and verified (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
		} else {
			b.WriteString("Implementation session complete — applied and verified.\n\n")
		}
	case proposed && state != nil && len(state.FilesChanged) > 0:
		b.WriteString(fmt.Sprintf("Implementation session complete — proposals submitted for approval (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
	default:
		b.WriteString("Implementation session complete — proposals submitted for approval.\n\n")
	}

	if state != nil && len(state.PreflightErrors) > 0 && !proposed {
		b.WriteString("Preflight issues encountered:\n")
		for _, e := range state.PreflightErrors {
			b.WriteString("- ")
			b.WriteString(e)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if state != nil && state.VerifyOutput != "" {
		b.WriteString("Verification:\n```\n")
		b.WriteString(truncateImplLog(state.VerifyOutput, 2000))
		b.WriteString("\n```\n\n")
	} else if state != nil && state.VerifySkipped && proposed {
		b.WriteString("Verification skipped (interactive trust — approve proposals to apply changes).\n\n")
	}
	b.WriteString(strings.TrimSpace(lastResponse))
	return strings.TrimSpace(b.String())
}

func truncateImplLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...(truncated)"
}

func extractChangedPathsFromResponse(response string) []string {
	var paths []string
	matches := fileChangeBlockRegex.FindAllStringSubmatch(response, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if d, err := parseFileChangeDirective(m[1]); err == nil && d.Path != "" {
			paths = append(paths, d.Path)
		}
	}
	return paths
}

func appendUnique(dst []string, add []string) []string {
	seen := make(map[string]bool, len(dst))
	for _, p := range dst {
		seen[p] = true
	}
	for _, p := range add {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		dst = append(dst, p)
	}
	return dst
}

const proposeFileEditToolName = "propose_file_edit"

func proposeFileEditToolDefinition() ai.ClaudeToolDefinition {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Relative file path under workspace root"},"content":{"type":"string","description":"Full new file content"},"operation":{"type":"string","description":"create, edit, or delete"}},"required":["path","content"]}`)
	return ai.ClaudeToolDefinition{
		Name:        proposeFileEditToolName,
		Description: "Propose a file create or edit in the shared workspace (submitted for approval)",
		InputSchema: schema,
	}
}

func (a *Agent) executeProposeFileEditTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		Path      string `json:"path"`
		Content   string `json:"content"`
		Operation string `json:"operation"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid propose_file_edit input: %w", err)
		}
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	op := strings.ToLower(strings.TrimSpace(args.Operation))
	if op == "delete" {
		return "", fmt.Errorf("delete not supported via propose_file_edit in v1")
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	var err error
	err = a.proposeFileChangePreferEditOrCreate(ctx, channel, path, args.Content, msg)
	if err != nil {
		return "", err
	}
	if st := implementationSessionStateFromContext(ctx); st != nil {
		st.ProposedCount++
		st.FilesChanged = appendUnique(st.FilesChanged, []string{path})
	}
	return fmt.Sprintf(`{"status":"proposed","path":%q}`, path), nil
}
