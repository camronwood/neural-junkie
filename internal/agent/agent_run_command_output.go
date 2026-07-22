package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const metadataAgentRunCommand = "agent_run_command"
const metadataMirrorTerminal = "mirror_terminal"

// Max chars of stdout/stderr embedded in command_output Content (LLM history reads Content, not metadata).
const maxAgentCommandOutputChars = 12_000

func parseReadFileToolInput(input json.RawMessage) string {
	var args map[string]interface{}
	if len(input) > 0 {
		_ = json.Unmarshal(input, &args)
	}
	for _, key := range []string{"path", "file_path", "target_path"} {
		if v, ok := args[key].(string); ok {
			if p := strings.TrimSpace(v); p != "" {
				return p
			}
		}
	}
	return ""
}

func parseRunCommandToolInput(input json.RawMessage) string {
	var args struct {
		Command string `json:"command"`
	}
	if len(input) > 0 {
		_ = json.Unmarshal(input, &args)
	}
	return strings.TrimSpace(args.Command)
}

func parseRunCommandMCPResult(result string) (exitCode int, stdout, stderr string) {
	result = strings.TrimSpace(result)
	if result == "" {
		return 1, "", "empty command result"
	}
	// MCP / agent tool failures are formatted as ERROR: … without exit_code=.
	// Never treat those as exit 0 — that posts false "success" command_output.
	lower := strings.ToLower(result)
	if strings.HasPrefix(result, "ERROR:") || strings.Contains(lower, "not allowlisted") {
		return 1, "", result
	}
	exitCode = 0
	if strings.HasPrefix(result, "exit_code=") {
		if i := strings.IndexByte(result, '\n'); i >= 0 {
			codeStr := strings.TrimPrefix(result[:i], "exit_code=")
			if n, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
				exitCode = n
			}
			result = result[i+1:]
		} else {
			codeStr := strings.TrimPrefix(result, "exit_code=")
			if n, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
				exitCode = n
				result = ""
			}
		}
	}
	stdout = result
	return exitCode, stdout, ""
}

func truncateCommandOutputBody(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max] + "\n... (truncated)"
}

// formatAgentRunCommandContent builds the chat/LLM-visible body for an agent-run shell command.
// Mirrors desktop formatCommandOutputContent so stdout/stderr survive into historyForGeneration.
func formatAgentRunCommandContent(agentName, command string, exitCode int, stdout, stderr string) string {
	who := strings.TrimSpace(agentName)
	if who == "" {
		who = "An agent"
	} else {
		who = "@" + who
	}
	status := "success"
	if exitCode != 0 {
		status = "failed"
	}
	var b strings.Builder
	b.WriteString(who)
	b.WriteString(" ran a terminal command.\n\n")
	b.WriteString("Command: `")
	b.WriteString(command)
	b.WriteString("`\n")
	b.WriteString(fmt.Sprintf("Exit code: %d (%s)\n", exitCode, status))
	if out := truncateCommandOutputBody(stdout, maxAgentCommandOutputChars); out != "" {
		b.WriteString("\nstdout:\n```\n")
		b.WriteString(out)
		b.WriteString("\n```\n")
	}
	if errOut := truncateCommandOutputBody(stderr, maxAgentCommandOutputChars); errOut != "" {
		b.WriteString("\nstderr:\n```\n")
		b.WriteString(errOut)
		b.WriteString("\n```\n")
	}
	if strings.TrimSpace(stdout) == "" && strings.TrimSpace(stderr) == "" {
		b.WriteString("\n(no output)\n")
	}
	return strings.TrimSpace(b.String())
}

// broadcastAgentRunCommandOutput posts command_output to the channel so humans and
// later LLM turns see what the agent ran (stdout/stderr in Content, not metadata-only).
func (a *Agent) broadcastAgentRunCommandOutput(msg *protocol.Message, command, mcpResult string) {
	if a == nil || a.Hub == nil || msg == nil {
		return
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	exitCode, stdout, stderr := parseRunCommandMCPResult(mcpResult)
	success := exitCode == 0
	cmdOut := protocol.CommandOutput{
		Command:  command,
		Plugin:   "shell",
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Success:  success,
	}
	raw, err := json.Marshal(cmdOut)
	if err != nil {
		return
	}
	content := formatAgentRunCommandContent(a.Info.Name, command, exitCode, stdout, stderr)
	out := protocol.NewMessage(protocol.MessageTypeCommandOutput, msg.Channel, a.Info, content)
	if out.Metadata == nil {
		out.Metadata = make(map[string]interface{})
	}
	out.Metadata["command_output"] = string(raw)
	out.Metadata[metadataAgentRunCommand] = true
	out.Metadata[metadataMirrorTerminal] = true
	out.ReplyTo = msg.ID
	go func() {
		if err := a.Hub.SendMessage(out); err != nil {
			log.Printf("[%s] broadcast run_command output: %v", a.Info.Name, err)
		}
	}()
}

// --- same-turn run_command dedupe ------------------------------------------------

type runCommandDedupeKey struct{}

type runCommandTurnDedupe struct {
	mu      sync.Mutex
	results map[string]string // normalized command -> MCP result text
}

// withRunCommandTurnDedupe attaches a per-turn cache so identical run_command calls
// in one tool-loop iteration reuse the first result instead of re-executing.
func withRunCommandTurnDedupe(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(runCommandDedupeKey{}).(*runCommandTurnDedupe); ok {
		return ctx
	}
	return context.WithValue(ctx, runCommandDedupeKey{}, &runCommandTurnDedupe{
		results: make(map[string]string),
	})
}

func runCommandTurnDedupeFrom(ctx context.Context) *runCommandTurnDedupe {
	if ctx == nil {
		return nil
	}
	d, _ := ctx.Value(runCommandDedupeKey{}).(*runCommandTurnDedupe)
	return d
}

// lookupOrStoreRunCommandResult returns a cached result for an identical command in this turn.
// ok=true means the command already ran; callers should not re-execute.
func lookupOrStoreRunCommandResult(ctx context.Context, command, result string, store bool) (cached string, ok bool) {
	d := runCommandTurnDedupeFrom(ctx)
	if d == nil {
		return "", false
	}
	key := normalizeImplCommand(command)
	if key == "" {
		return "", false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if prev, hit := d.results[key]; hit {
		return prev, true
	}
	if store && result != "" {
		d.results[key] = result
	}
	return "", false
}

func storeRunCommandTurnResult(ctx context.Context, command, result string) {
	d := runCommandTurnDedupeFrom(ctx)
	if d == nil {
		return
	}
	key := normalizeImplCommand(command)
	if key == "" {
		return
	}
	d.mu.Lock()
	d.results[key] = result
	d.mu.Unlock()
}
