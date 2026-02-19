package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitHubExecutor handles execution of GitHub CLI commands
type GitHubExecutor struct {
	timeout time.Duration
}

// NewGitHubExecutor creates a new GitHub CLI executor
func NewGitHubExecutor() *GitHubExecutor {
	return &GitHubExecutor{
		timeout: DefaultTimeout,
	}
}

// CheckGHCLI validates that gh CLI is installed and authenticated
func (ge *GitHubExecutor) CheckGHCLI() error {
	// Check if gh is installed
	cmd := exec.Command("which", "gh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not installed. Install from https://cli.github.com/")
	}

	// Check if authenticated
	cmd = exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not authenticated. Run 'gh auth login' to authenticate")
	}

	return nil
}

// ExecuteGitHubCommand executes a GitHub CLI command
func (ge *GitHubExecutor) ExecuteGitHubCommand(subCmd string, args []string) (*CommandResult, error) {
	start := time.Now()

	// Validate gh CLI is available
	if err := ge.CheckGHCLI(); err != nil {
		return &CommandResult{
			Command:   fmt.Sprintf("gh %s %s", subCmd, strings.Join(args, " ")),
			Plugin:    "gh",
			SubCmd:    subCmd,
			Args:      args,
			ExitCode:  1,
			Stderr:    err.Error(),
			Success:   false,
			Duration:  time.Since(start),
			Timestamp: start,
		}, err
	}

	// Build gh command based on subcommand
	ghArgs := ge.buildGHArgs(subCmd, args)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ge.timeout)
	defer cancel()

	// Build full command string for display
	fullCmd := fmt.Sprintf("gh %s", strings.Join(ghArgs, " "))

	// Create command
	cmd := exec.CommandContext(ctx, "gh", ghArgs...)

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
		Plugin:    "gh",
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

// buildGHArgs builds the gh CLI arguments based on subcommand and args
func (ge *GitHubExecutor) buildGHArgs(subCmd string, args []string) []string {
	ghArgs := []string{}

	switch subCmd {
	// Repository operations
	case "repo-view":
		ghArgs = append(ghArgs, "repo", "view")
		ghArgs = append(ghArgs, args...)
	case "repo-list":
		ghArgs = append(ghArgs, "repo", "list")
		ghArgs = append(ghArgs, args...)
	case "repo-clone":
		ghArgs = append(ghArgs, "repo", "clone")
		ghArgs = append(ghArgs, args...)
	case "repo-fork":
		ghArgs = append(ghArgs, "repo", "fork")
		ghArgs = append(ghArgs, args...)
	case "repo-create":
		ghArgs = append(ghArgs, "repo", "create")
		ghArgs = append(ghArgs, args...)

	// Issue operations
	case "issue-list":
		ghArgs = append(ghArgs, "issue", "list")
		ghArgs = append(ghArgs, args...)
	case "issue-view":
		ghArgs = append(ghArgs, "issue", "view")
		ghArgs = append(ghArgs, args...)
	case "issue-create":
		ghArgs = append(ghArgs, "issue", "create")
		ghArgs = append(ghArgs, args...)
	case "issue-close":
		ghArgs = append(ghArgs, "issue", "close")
		ghArgs = append(ghArgs, args...)
	case "issue-reopen":
		ghArgs = append(ghArgs, "issue", "reopen")
		ghArgs = append(ghArgs, args...)
	case "issue-comment":
		ghArgs = append(ghArgs, "issue", "comment")
		ghArgs = append(ghArgs, args...)

	// PR operations
	case "pr-list":
		ghArgs = append(ghArgs, "pr", "list")
		ghArgs = append(ghArgs, args...)
	case "pr-view":
		ghArgs = append(ghArgs, "pr", "view")
		ghArgs = append(ghArgs, args...)
	case "pr-create":
		ghArgs = append(ghArgs, "pr", "create")
		ghArgs = append(ghArgs, args...)
	case "pr-checkout":
		ghArgs = append(ghArgs, "pr", "checkout")
		ghArgs = append(ghArgs, args...)
	case "pr-review":
		ghArgs = append(ghArgs, "pr", "review")
		ghArgs = append(ghArgs, args...)
	case "pr-merge":
		ghArgs = append(ghArgs, "pr", "merge")
		ghArgs = append(ghArgs, args...)
	case "pr-close":
		ghArgs = append(ghArgs, "pr", "close")
		ghArgs = append(ghArgs, args...)
	case "pr-diff":
		ghArgs = append(ghArgs, "pr", "diff")
		ghArgs = append(ghArgs, args...)

	// Search operations
	case "search-code":
		ghArgs = append(ghArgs, "search", "code")
		ghArgs = append(ghArgs, args...)
	case "search-repos":
		ghArgs = append(ghArgs, "search", "repos")
		ghArgs = append(ghArgs, args...)
	case "search-issues":
		ghArgs = append(ghArgs, "search", "issues")
		ghArgs = append(ghArgs, args...)
	case "search-prs":
		ghArgs = append(ghArgs, "search", "prs")
		ghArgs = append(ghArgs, args...)

	// Status and auth
	case "auth-status":
		ghArgs = append(ghArgs, "auth", "status")
		ghArgs = append(ghArgs, args...)
	case "status":
		ghArgs = append(ghArgs, "status")
		ghArgs = append(ghArgs, args...)

	// Workflow operations
	case "workflow-list":
		ghArgs = append(ghArgs, "workflow", "list")
		ghArgs = append(ghArgs, args...)
	case "workflow-view":
		ghArgs = append(ghArgs, "workflow", "view")
		ghArgs = append(ghArgs, args...)
	case "workflow-run":
		ghArgs = append(ghArgs, "workflow", "run")
		ghArgs = append(ghArgs, args...)

	// Run operations (GitHub Actions)
	case "run-list":
		ghArgs = append(ghArgs, "run", "list")
		ghArgs = append(ghArgs, args...)
	case "run-view":
		ghArgs = append(ghArgs, "run", "view")
		ghArgs = append(ghArgs, args...)
	case "run-watch":
		ghArgs = append(ghArgs, "run", "watch")
		ghArgs = append(ghArgs, args...)

	default:
		// Unknown subcommand, pass through as-is
		ghArgs = append(ghArgs, subCmd)
		ghArgs = append(ghArgs, args...)
	}

	return ghArgs
}
