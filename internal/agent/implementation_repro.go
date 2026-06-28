package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	userMentionedCommandRE = regexp.MustCompile(`(?i)(?:^|\s|\$)(npm run [a-z0-9:_-]+|npm test|npm install [^\s]+|npx [^\s]+|make [a-z0-9_-]+|go test(?:\s+\./\.\.\.)?|go build(?:\s+\./\.\.\.)?|cargo test|cargo build|python -m pytest(?:\s+-q)?|pytest(?:\s+-q)?)`)
	makeFailedTargetRE     = regexp.MustCompile(`(?i)no rule to make target ['"]([^'"]+)['"]`)
)

const fixLikeMaxToolIterations = 25
const fixLikeMaxFilesPerMessage = 6

// inferReproCommand picks the canonical repro command for a fix-like session.
func inferReproCommand(wsPath string, manifest *StackManifest, userText string) string {
	if target := extractMakefileTargetFromError(userText); target != "" {
		return "make " + target
	}
	if cmd := extractUserMentionedCommand(userText); cmd != "" {
		return cmd
	}
	if manifest != nil {
		if cmds := manifest.DefaultReproCommands(userText); len(cmds) > 0 {
			return cmds[0]
		}
	}
	if wsPath != "" {
		if cmds := bootFixDiagnosticCommands(wsPath, manifest); len(cmds) > 0 {
			return cmds[0]
		}
	}
	return ""
}

func extractUserMentionedCommand(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if m := userMentionedCommandRE.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(strings.TrimPrefix(m[1], "$"))
	}
	return ""
}

func extractMakefileTargetFromError(text string) string {
	if m := makeFailedTargetRE.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// runReproBootstrap runs the repro command once, applies playbooks, and stores repro state.
func (a *Agent) runReproBootstrap(
	ctx context.Context,
	msg *protocol.Message,
	state *ImplementationSessionState,
	wsPath string,
) (bool, string) {
	if a == nil || msg == nil || state == nil || wsPath == "" || !state.FixLikeIntent {
		return false, ""
	}
	state.DiagnosticBootstrapDone = true
	state.ReproBootstrapActive = true
	defer func() { state.ReproBootstrapActive = false }()

	reproCmd := strings.TrimSpace(state.ReproCommand)
	if reproCmd == "" {
		reproCmd = inferReproCommand(wsPath, state.StackManifest, msg.Content)
	}
	if reproCmd == "" {
		return false, formatReproDiagnosticNote(state)
	}
	state.ReproCommand = reproCmd

	a.bootstrapReproReads(ctx, msg, state, wsPath)

	a.sendThinkingActivity(msg, protocol.ThinkingActivityImplementation, "Running "+reproCmd+"…")
	result, err := a.runReproCommand(ctx, msg, state, reproCmd)
	if err != nil {
		note := formatReproDiagnosticNote(state)
		return false, note
	}
	exitCode, _, _ := parseRunCommandMCPResult(result)
	state.RecordCommandRun(reproCmd, exitCode, result)
	state.ReproExitCode = exitCode
	state.ReproOutput = result
	state.clearReproBootstrapFailures()

	if !shouldSkipDuplicateCommandBroadcast(msg.Channel, a.Info.ID, reproCmd, result) {
		a.broadcastAgentRunCommandOutput(msg, reproCmd, result)
	}

	sig := commandOutputMatchesPlaybook(result)
	if sig == "" && messageHasBootOrBuildError(msg.Content) {
		sig = commandOutputMatchesPlaybook(msg.Content)
	}
	if sig != "" {
		channel := msg.Channel
		if channel == "" {
			channel = "general"
		}
		if ok, _ := a.attemptCommandFailurePlaybook(ctx, msg, wsPath, channel, sig, state); ok {
			logReproDiagnostic(a, "playbook_applied", sig, reproCmd)
			return true, ""
		}
	}

	note := formatReproDiagnosticNote(state)
	logReproDiagnostic(a, "repro_complete", "", reproCmd)
	return false, note
}

func (a *Agent) bootstrapReproReads(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState, wsPath string) {
	_ = ctx
	_ = msg
	for _, rel := range bootFixBootstrapReadPaths(wsPath, state.StackManifest) {
		a.bootstrapReadWorkspaceFile(state, wsPath, rel)
	}
}

func (a *Agent) runReproCommand(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState, cmd string) (string, error) {
	if state != nil {
		if err := state.ShouldBlockRunCommand(cmd); err != nil {
			return "", err
		}
	}
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return "", fmt.Errorf("workspace MCP not available")
	}
	toolCtx := a.contextWithWorkspaceBackend(ctx, msg)
	toolCtx = shared.ContextWithImplementationSession(toolCtx, true)
	toolCtx = shared.ContextWithBootFixDiagnostic(toolCtx, true)
	timeout := shared.BootFixRunCommandTimeout
	if shared.IsDevServerCommand(cmd) {
		timeout = 15 * time.Second
	}
	toolCtx = shared.ContextWithRunCommandTimeout(toolCtx, timeout)
	toolCtx = attachImplSessionCommandPolicy(toolCtx, state)
	toolCtx = shared.ContextWithRunCommandProgress(toolCtx, func(line string) {
		if a == nil || msg == nil {
			return
		}
		detail := strings.TrimSpace(line)
		if detail == "" {
			return
		}
		if len(detail) > 100 {
			detail = detail[:100] + "…"
		}
		a.sendThinkingActivity(msg, protocol.ThinkingActivityImplementation, cmd+": "+detail)
	})
	input, _ := json.Marshal(map[string]string{"command": cmd})
	return executeMCPTool(toolCtx, mcpServer, "run_command", input)
}

func (a *Agent) runReproVerify(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) (output string, failed bool, skipped bool) {
	if state == nil || strings.TrimSpace(state.ReproCommand) == "" {
		return a.runImplementationVerify(ctx, msg, state)
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return "", false, true
	}
	cmd := strings.TrimSpace(state.ReproCommand)
	a.sendThinkingActivity(msg, protocol.ThinkingActivityVerifying, cmd)
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return "", false, true
	}
	toolCtx := a.contextWithWorkspaceBackend(ctx, msg)
	toolCtx = shared.ContextWithImplementationSession(toolCtx, true)
	toolCtx = attachImplSessionCommandPolicy(toolCtx, state)
	input, _ := json.Marshal(map[string]string{"command": cmd})
	result, err := executeMCPTool(toolCtx, mcpServer, "run_command", input)
	var combined strings.Builder
	combined.WriteString("$ ")
	combined.WriteString(cmd)
	combined.WriteString("\n")
	if err != nil {
		combined.WriteString(err.Error())
		state.ReproExitCode = 1
		state.ReproOutput = combined.String()
		return combined.String(), true, false
	}
	combined.WriteString(result)
	exitCode, _, _ := parseRunCommandMCPResult(result)
	state.RecordCommandRun(cmd, exitCode, result)
	state.ReproExitCode = exitCode
	state.ReproOutput = combined.String()
	failed = exitCode != 0
	return combined.String(), failed, false
}

func formatReproDiagnosticNote(state *ImplementationSessionState) string {
	if state == nil {
		return "Fix session: use repro command output to propose file edits with search_replace or propose_file_edit."
	}
	var b strings.Builder
	b.WriteString("Repro diagnostics completed.\n")
	if cmd := strings.TrimSpace(state.ReproCommand); cmd != "" {
		b.WriteString("Repro command: ")
		b.WriteString(cmd)
		b.WriteString("\n")
	} else if cmd := strings.TrimSpace(state.LastFailedCommand); cmd != "" {
		b.WriteString("Last command: ")
		b.WriteString(cmd)
		b.WriteString("\n")
	}
	if out := strings.TrimSpace(state.LastCommandOutput()); out != "" {
		b.WriteString("```\n")
		b.WriteString(truncateImplLog(out, 2500))
		b.WriteString("\n```\n")
	}
	if sig := commandOutputMatchesPlaybook(state.LastCommandOutput()); sig != "" {
		b.WriteString("Failure class: ")
		b.WriteString(sig)
		b.WriteString(" — ")
		b.WriteString(bootFixFailureClassHint(sig))
		b.WriteString("\n")
	} else {
		b.WriteString("Use the output above to propose concrete file edits. Advice-only replies do not complete this session.\n")
	}
	return b.String()
}

func formatFixInterimProgress(state *ImplementationSessionState) string {
	if state == nil {
		return "Repro finished — analyzing output…"
	}
	var b strings.Builder
	b.WriteString("Repro finished")
	if cmd := strings.TrimSpace(state.ReproCommand); cmd != "" {
		b.WriteString(" (`")
		b.WriteString(cmd)
		b.WriteString("`)")
	} else if cmd := strings.TrimSpace(state.LastFailedCommand); cmd != "" {
		b.WriteString(" (`")
		b.WriteString(cmd)
		b.WriteString("`)")
	}
	if sig := commandOutputMatchesPlaybook(state.LastCommandOutput()); sig != "" {
		b.WriteString(" — detected ")
		b.WriteString(sig)
	}
	b.WriteString(". Analyzing output and preparing a fix…")
	return b.String()
}

func logReproDiagnostic(a *Agent, event, sig, cmd string) {
	if a == nil {
		return
	}
	if sig != "" {
		log.Printf("[%s] repro_diagnostic(event=%s,sig=%s,cmd=%s)", a.Info.Name, event, sig, cmd)
		return
	}
	log.Printf("[%s] repro_diagnostic(event=%s,cmd=%s)", a.Info.Name, event, cmd)
}

func completeFixDiagnoseFromRepro(state *ImplementationSessionState) {
	if state == nil || !state.FixLikeIntent {
		return
	}
	if sig := commandOutputMatchesPlaybook(state.LastCommandOutput()); sig != "" {
		state.DiagnosePhaseComplete = true
	}
}

func fixLikePostDiscoverRepairNote(state *ImplementationSessionState) string {
	if state != nil && strings.TrimSpace(state.LastCommandOutput()) != "" {
		return formatReproDiagnosticNote(state)
	}
	return "Fix session: run the repro command to capture errors, then propose_file_edit or search_replace to fix the root cause."
}

func npmModuleInstalled(wsPath, module string) bool {
	module = strings.TrimSpace(module)
	if module == "" || wsPath == "" {
		return false
	}
	pkgRoot := module
	if strings.HasPrefix(pkgRoot, "@") {
		parts := strings.SplitN(pkgRoot, "/", 3)
		if len(parts) >= 2 {
			pkgRoot = parts[0] + "/" + parts[1]
		}
	} else if i := strings.IndexByte(pkgRoot, '/'); i > 0 {
		pkgRoot = pkgRoot[:i]
	}
	_, err := os.Stat(filepath.Join(wsPath, "node_modules", filepath.FromSlash(pkgRoot)))
	return err == nil
}

func implSessionLimitsForState(msg *protocol.Message, state *ImplementationSessionState) (maxToolIter, maxEditRounds, maxFiles int) {
	maxToolIter, maxEditRounds, maxFiles = implSessionLimits(msg)
	if state != nil && state.FixLikeIntent {
		if maxToolIter > fixLikeMaxToolIterations {
			maxToolIter = fixLikeMaxToolIterations
		}
		if maxFiles > fixLikeMaxFilesPerMessage {
			maxFiles = fixLikeMaxFilesPerMessage
		}
	}
	return maxToolIter, maxEditRounds, maxFiles
}

func (a *Agent) sendInterimFixUpdate(msg *protocol.Message, text string) {
	if a == nil || a.Hub == nil || msg == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	update := protocol.NewMessage(protocol.MessageTypeAnswer, msg.Channel, a.Info, text)
	update.ReplyTo = msg.ID
	if msg.IsInThread() {
		update.ThreadID = msg.ThreadID
		update.IsThreadReply = true
	}
	_ = a.Hub.SendMessage(update)
}

func (a *Agent) runVerifyForState(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) (output string, failed bool, skipped bool) {
	if state != nil && state.FixLikeIntent && strings.TrimSpace(state.ReproCommand) != "" {
		return a.runReproVerify(ctx, msg, state)
	}
	return a.runImplementationVerify(ctx, msg, state)
}

func fixLikeSessionSucceeded(state *ImplementationSessionState) bool {
	if state == nil || !state.FixLikeIntent {
		return false
	}
	if strings.TrimSpace(state.ReproCommand) == "" {
		return !state.VerifyFailed && !state.VerifySkipped
	}
	return state.ReproExitCode == 0
}
