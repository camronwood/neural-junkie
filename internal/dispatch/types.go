package dispatch

import (
	"time"
)

// CommandResult represents the result of executing a dispatch command
type CommandResult struct {
	Command   string        // Full command string
	Plugin    string        // dispatch plugin name
	SubCmd    string        // Subcommand (e.g., "list", "create")
	Args      []string      // Command arguments
	ExitCode  int           // Exit code
	Stdout    string        // Standard output
	Stderr    string        // Standard error
	Duration  time.Duration // Execution time
	Success   bool          // Whether command succeeded
	Timestamp time.Time     // When command was executed
}

// PendingApproval represents a command awaiting approval
type PendingApproval struct {
	ID          string    // Unique approval ID
	UserID      string    // User who requested the command
	Username    string    // User's display name
	Channel     string    // Channel where command was requested
	Command     string    // Full command string
	Plugin      string    // dispatch plugin
	SubCmd      string    // Subcommand
	Args        []string  // Arguments
	RequestedAt time.Time // When approval was requested
	ExpiresAt   time.Time // When approval expires
}

// CommandDefinition defines a dispatch command's properties
type CommandDefinition struct {
	Plugin      string   // Plugin name (e.g., "subenv", "aws")
	SubCommand  string   // Subcommand name (e.g., "list", "create")
	Description string   // Human-readable description
	ReadOnly    bool     // Whether the command is read-only
	Examples    []string // Example usages
}
