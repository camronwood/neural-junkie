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
		a.sendThinkingActivity(msg, protocol.ThinkingActivityVerifying, "boot diagnostic: "+cmd)
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
	if manifest != nil && fileExists(filepath.Join(wsPath, "package.json")) {
		if npmScriptExists(wsPath, "build") {
			cmds = append(cmds, "npm run build")
		} else if cmd := shared.TypeScriptCheckShellCommand(wsPath); cmd != "" {
			cmds = append(cmds, cmd)
		} else if npmScriptExists(wsPath, "typecheck") {
			cmds = append(cmds, "npm run typecheck")
		}
	}
	if fileExists(filepath.Join(wsPath, "go.mod")) {
		cmds = append(cmds, "go build ./...")
	}
	return cmds
}

func npmScriptExists(wsPath, script string) bool {
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
	_, ok := pkg.Scripts[script]
	return ok
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
	toolCtx = shared.ContextWithBootFixDiagnostic(toolCtx, true)
	toolCtx = shared.ContextWithRunCommandTimeout(toolCtx, shared.BootFixRunCommandTimeout)
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
		a.sendThinkingActivity(msg, protocol.ThinkingActivityVerifying, "boot diagnostic: "+detail)
	})
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
		b.WriteString(" — ")
		b.WriteString(bootFixFailureClassHint(sig))
		b.WriteString("\n")
	} else {
		b.WriteString("Use the output above to propose concrete file edits. Advice-only replies do not complete this session.\n")
	}
	return b.String()
}

func bootFixFailureClassHint(sig string) string {
	switch sig {
	case "missing_start_all_target":
		return "ship the repair with search_replace or propose_file_edit"
	case "missing_npm_module":
		return "add the missing dependency to package.json and run npm install, or remove the import"
	case "vite_strict_port_conflict":
		return "align vite strictPort/devServer.port with tauri devPath or free the port"
	case "dev_server_timeout":
		return "do not run long-lived dev servers for diagnostics — fix compile/build errors first"
	case "tauri_vite_port_mismatch":
		return "align Tauri devPath with the Vite dev server port"
	default:
		return "ship the repair with search_replace or propose_file_edit"
	}
}

func bootFixPostDiscoverRepairNote(state *ImplementationSessionState) string {
	if state != nil && strings.TrimSpace(state.LastCommandOutput()) != "" {
		return formatBootFixDiagnosticNote(state)
	}
	return "Boot-fix: run run_command (npm run build or tsc --noEmit) to capture compile errors, then propose_file_edit or search_replace to fix the root cause."
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

func formatBootFixInterimProgress(state *ImplementationSessionState) string {
	if state == nil {
		return "Boot diagnostics finished — analyzing build output…"
	}
	var b strings.Builder
	b.WriteString("Boot diagnostics finished")
	if cmd := strings.TrimSpace(state.LastFailedCommand); cmd != "" {
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
