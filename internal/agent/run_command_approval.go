package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/mcp/workspace"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// runCommandApprovalHub is implemented by the in-process hub for allowlist prompts.
type runCommandApprovalHub interface {
	RequestRunCommandApproval(agentID, agentName, channel, command string) (approved bool, always bool, err error)
}

// maybeApproveRunCommand prompts the user when a command is not allowlisted
// (but not hard-denied). On approval it grants a one-shot context allow and
// optionally persists the command to Security.RunCommandAllowExtra.
func (a *Agent) maybeApproveRunCommand(ctx context.Context, msg *protocol.Message, command string) (context.Context, error) {
	command = strings.Join(strings.Fields(strings.TrimSpace(command)), " ")
	if command == "" {
		return ctx, fmt.Errorf("command is required")
	}
	if workspace.CommandHardDenied(command) {
		return ctx, fmt.Errorf("command blocked for safety: %s", command)
	}
	if workspace.CommandAllowedForContext(ctx, command) {
		return ctx, nil
	}
	if a == nil || a.Hub == nil || msg == nil {
		return ctx, fmt.Errorf("command not allowlisted: %s", command)
	}

	channel := strings.TrimSpace(msg.Channel)
	approved := false
	always := false
	var err error
	if hub, ok := a.Hub.(runCommandApprovalHub); ok {
		approved, always, err = hub.RequestRunCommandApproval(a.Info.ID, a.Info.Name, channel, command)
	} else {
		approved, err = a.Hub.RequestToolApproval(a.Info.ID, a.Info.Name, channel, "run_command", map[string]interface{}{
			"command": command,
			"reason":  "not_allowlisted",
		})
	}
	if err != nil {
		return ctx, err
	}
	if !approved {
		return ctx, fmt.Errorf("command not allowlisted (user declined): %s", command)
	}
	if always {
		if cfg := mcpAppConfig(); cfg != nil {
			if _, addErr := cfg.AddRunCommandAllowExtra(command); addErr == nil {
				_ = cfg.Save()
			}
		}
	}
	ctx = shared.ContextWithRunCommandUserAllow(ctx, command)
	if always {
		ctx = shared.ContextWithRunCommandExtraAllows(ctx, mcpAppConfig().RunCommandAllowExtra())
	}
	return ctx, nil
}
