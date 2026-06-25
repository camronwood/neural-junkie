package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const metadataAgentRunCommand = "agent_run_command"
const metadataMirrorTerminal = "mirror_terminal"

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
	exitCode = 0
	if strings.HasPrefix(result, "exit_code=") {
		if i := strings.IndexByte(result, '\n'); i >= 0 {
			codeStr := strings.TrimPrefix(result[:i], "exit_code=")
			if n, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
				exitCode = n
			}
			result = result[i+1:]
		}
	}
	stdout = result
	return exitCode, stdout, ""
}

// broadcastAgentRunCommandOutput posts command_output to the channel so humans see
// what the agent ran. Desktop mirrors this into the terminal panel when mirror_terminal is set.
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
	status := "success"
	if !success {
		status = "failed"
	}
	content := fmt.Sprintf(
		"@%s ran `%s` (exit %d, %s).",
		a.Info.Name, command, exitCode, status,
	)
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
