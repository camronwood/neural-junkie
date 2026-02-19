package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default timeout for command execution
	DefaultTimeout = 30 * time.Second
)

// Executor handles execution of dispatch CLI commands
type Executor struct {
	timeout        time.Duration
	githubExecutor *GitHubExecutor
}

// NewExecutor creates a new command executor with default timeout
func NewExecutor() *Executor {
	return &Executor{
		timeout:        DefaultTimeout,
		githubExecutor: NewGitHubExecutor(),
	}
}

// NewExecutorWithTimeout creates a new executor with custom timeout
func NewExecutorWithTimeout(timeout time.Duration) *Executor {
	return &Executor{
		timeout: timeout,
	}
}

// IsDispatchInstalled checks if the dispatch CLI is available
func (e *Executor) IsDispatchInstalled() bool {
	cmd := exec.Command("which", "dispatch")
	return cmd.Run() == nil
}

// ExecuteCommand executes a dispatch command and returns the result
func (e *Executor) ExecuteCommand(ctx context.Context, plugin, subCmd string, args []string) (*CommandResult, error) {
	// Route GitHub CLI commands through GitHubExecutor
	if plugin == "gh" {
		return e.githubExecutor.ExecuteGitHubCommand(subCmd, args)
	}

	start := time.Now()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Build full command string
	fullCmd := fmt.Sprintf("dispatch %s %s", plugin, subCmd)
	if len(args) > 0 {
		fullCmd = fmt.Sprintf("%s %s", fullCmd, strings.Join(args, " "))
	}

	// Prepare command arguments
	cmdArgs := []string{plugin, subCmd}
	cmdArgs = append(cmdArgs, args...)

	// Create command
	cmd := exec.CommandContext(timeoutCtx, "dispatch", cmdArgs...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	duration := time.Since(start)

	// Determine exit code
	exitCode := 0
	success := true
	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or context cancelled
			exitCode = -1
		}
	}

	result := &CommandResult{
		Command:   fullCmd,
		Plugin:    plugin,
		SubCmd:    subCmd,
		Args:      args,
		ExitCode:  exitCode,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Duration:  duration,
		Success:   success,
		Timestamp: start,
	}

	return result, nil
}

// ExecuteCommandString executes a full dispatch command string
// Example: "dispatch subenv list" -> ExecuteCommand("subenv", "list", [])
func (e *Executor) ExecuteCommandString(ctx context.Context, cmdString string) (*CommandResult, error) {
	parts := strings.Fields(cmdString)

	// Remove "dispatch" prefix if present
	if len(parts) > 0 && parts[0] == "dispatch" {
		parts = parts[1:]
	}

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid dispatch command: must have at least plugin and subcommand")
	}

	plugin := parts[0]
	subCmd := parts[1]
	args := []string{}
	if len(parts) > 2 {
		args = parts[2:]
	}

	return e.ExecuteCommand(ctx, plugin, subCmd, args)
}
