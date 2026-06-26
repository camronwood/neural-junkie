package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// runBootFixDiagnosticBootstrap seeds reads, runs allowlisted diagnostics, and applies
// playbooks only when command output proves a failure class (Fix Loop v2).
// Returns (playbookApplied, repairNote).
func (a *Agent) runBootFixDiagnosticBootstrap(
	ctx context.Context,
	msg *protocol.Message,
	state *ImplementationSessionState,
	wsPath string,
) (bool, string) {
	if a == nil || msg == nil || state == nil || wsPath == "" || !state.BootFixIntent {
		return false, ""
	}
	state.DiagnosticBootstrapDone = true

	a.bootstrapBootFixReads(ctx, msg, state, wsPath)

	for _, cmd := range bootFixDiagnosticCommands(wsPath, state.StackManifest) {
		result, err := a.runBootFixDiagnosticCommand(ctx, msg, state, cmd)
		if err != nil {
			continue
		}
		exitCode, _, _ := parseRunCommandMCPResult(result)
		state.RecordCommandRun(cmd, exitCode, result)
		if !shouldSkipDuplicateCommandBroadcast(msg.Channel, a.Info.ID, cmd, result) {
			a.broadcastAgentRunCommandOutput(msg, cmd, result)
		}
		sig := commandOutputMatchesPlaybook(result)
		if sig == "" {
			continue
		}
		channel := msg.Channel
		if channel == "" {
			channel = "general"
		}
		if ok, _ := a.attemptCommandFailurePlaybook(ctx, msg, wsPath, channel, sig, state); ok {
			logBootFixDiagnostic(a, "playbook_applied", sig, cmd)
			return true, ""
		}
	}

	note := formatBootFixDiagnosticNote(state)
	logBootFixDiagnostic(a, "diagnostics_complete", "", state.LastFailedCommand)
	return false, note
}

func (a *Agent) bootstrapBootFixReads(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState, wsPath string) {
	_ = ctx
	_ = msg
	for _, rel := range bootFixBootstrapReadPaths(wsPath, state.StackManifest) {
		a.bootstrapReadWorkspaceFile(state, wsPath, rel)
	}
}

func bootFixBootstrapReadPaths(wsPath string, manifest *StackManifest) []string {
	var paths []string
	if fileExists(filepath.Join(wsPath, "Makefile")) {
		paths = append(paths, "Makefile")
	}
	if manifest != nil && manifest.HasReact && fileExists(filepath.Join(wsPath, "package.json")) {
		paths = append(paths, "package.json")
	}
	if fileExists(filepath.Join(wsPath, "scripts", "start-all.sh")) {
		paths = append(paths, "scripts/start-all.sh")
	}
	return paths
}

func bootFixDiagnosticCommands(wsPath string, manifest *StackManifest) []string {
	var cmds []string
	if fileExists(filepath.Join(wsPath, "scripts", "start-all.sh")) && fileExists(filepath.Join(wsPath, "Makefile")) {
		cmds = append(cmds, "make start-all")
	}
	if manifest != nil && manifest.HasReact && fileExists(filepath.Join(wsPath, "package.json")) {
		cmds = append(cmds, "npm run build")
	}
	return cmds
}

func (a *Agent) bootstrapReadWorkspaceFile(state *ImplementationSessionState, wsPath, rel string) {
	if state == nil || wsPath == "" || rel == "" {
		return
	}
	if _, err := os.ReadFile(filepath.Join(wsPath, rel)); err != nil {
		return
	}
	state.RecordReadPath(rel)
	state.recordDiscoverTool("read_file")
}

func (a *Agent) runBootFixDiagnosticCommand(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState, cmd string) (string, error) {
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
	toolCtx = attachImplSessionCommandPolicy(toolCtx, state)
	input, _ := json.Marshal(map[string]string{"command": cmd})
	return executeMCPTool(toolCtx, mcpServer, "run_command", input)
}

func formatBootFixDiagnosticNote(state *ImplementationSessionState) string {
	if state == nil {
		return "Boot-fix: use diagnostic command output to propose file edits with search_replace or propose_file_edit."
	}
	var b strings.Builder
	b.WriteString("Boot-fix diagnostics completed.\n")
	if cmd := strings.TrimSpace(state.LastFailedCommand); cmd != "" {
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
		b.WriteString(" — ship the repair with search_replace or propose_file_edit.\n")
	} else {
		b.WriteString("Use the output above to propose concrete file edits. Advice-only replies do not complete this session.\n")
	}
	return b.String()
}

func bootFixPostDiscoverRepairNote(state *ImplementationSessionState) string {
	if state != nil && strings.TrimSpace(state.LastCommandOutput()) != "" {
		return formatBootFixDiagnosticNote(state)
	}
	return "Boot-fix: run run_command (make start-all or npm run build) if needed, then propose_file_edit or search_replace to fix the root cause."
}

func logBootFixDiagnostic(a *Agent, event, sig, cmd string) {
	if a == nil {
		return
	}
	if sig != "" {
		log.Printf("[%s] bootfix_diagnostic(event=%s,sig=%s,cmd=%s)", a.Info.Name, event, sig, cmd)
		return
	}
	log.Printf("[%s] bootfix_diagnostic(event=%s,cmd=%s)", a.Info.Name, event, cmd)
}

func playbookSignatureFromCommandEvidence(content string) string {
	return commandOutputMatchesPlaybook(content)
}
