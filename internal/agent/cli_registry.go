package agent

import (
	"sort"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CLIAgentConfig describes a CLI tool that can be wrapped as an agent.
type CLIAgentConfig struct {
	Type         string   // Registry key: "cursor", "gemini", "claude", "copilot"
	Command      string   // Binary name on PATH: "agent", "gemini", "claude", "copilot"
	ProviderName string   // "cursor-cli", "gemini-cli", etc.
	ModelName    string   // "cursor-agent", "gemini-agent", etc.
	DefaultName  string   // Default agent display name
	BaseArgs     []string // Provider base args
	Expertise    []string // Agent expertise list
	EnvVars      []string // Env var names to forward (e.g. ["CURSOR_API_KEY"])
	WorkDirEnv   string   // Env var for work dir override (e.g. "CURSOR_WORK_DIR")
	ApprovalMode string   // "interactive", "auto_edit", "yolo", or ""
	InstallHint  string   // Help text if binary not found
	JoinMessage  string   // Message sent to channel on agent join
}

var cliAgentRegistry = map[string]CLIAgentConfig{
	"cursor": {
		Type:         "cursor",
		Command:      "agent",
		ProviderName: "cursor-cli",
		ModelName:    "cursor-agent",
		DefaultName:  "Cursor",
		BaseArgs:     []string{"-p", "--output-format", "text"},
		Expertise: []string{
			"Code Generation", "Refactoring", "Code Review",
			"Codebase Analysis", "Bug Fixing", "Testing",
			"Full-Stack Development", "Architecture",
			"File Operations", "Shell Commands",
		},
		EnvVars:      []string{"CURSOR_API_KEY"},
		WorkDirEnv:   "CURSOR_WORK_DIR",
		ApprovalMode: "",
		InstallHint:  "Install with: curl https://cursor.com/install -fsS | bash",
		JoinMessage:  "Cursor CLI agent online. I can analyze codebases, generate code, refactor, and run shell commands using Cursor's agent capabilities. @mention me to get started.",
	},
	"gemini": {
		Type:         "gemini",
		Command:      "gemini",
		ProviderName: "gemini-cli",
		ModelName:    "gemini-2.5-flash",
		DefaultName:  "Gemini",
		BaseArgs:     []string{"--output-format", "text", "-p"},
		Expertise: []string{
			"Code Generation", "Code Review", "Multimodal Analysis",
			"Codebase Analysis", "Architecture", "Refactoring",
			"Testing", "Documentation",
			"File Operations", "Shell Commands",
		},
		EnvVars:      nil,
		WorkDirEnv:   "GEMINI_WORK_DIR",
		ApprovalMode: "interactive",
		InstallHint:  "Install with: npm install -g @google/gemini-cli",
		JoinMessage:  "Gemini CLI agent online. I can analyze codebases, generate code, review, and run shell commands using Google's Gemini agent. @mention me to get started.",
	},
	"claude": {
		Type:         "claude",
		Command:      "claude",
		ProviderName: "claude-cli",
		ModelName:    "claude-agent",
		DefaultName:  "Claude",
		BaseArgs:     []string{"-p"},
		Expertise: []string{
			"Code Generation", "Code Review", "Architecture",
			"Codebase Analysis", "Refactoring", "Bug Fixing",
			"Testing", "Documentation",
			"File Operations", "Shell Commands",
		},
		EnvVars:      []string{"ANTHROPIC_API_KEY"},
		WorkDirEnv:   "CLAUDE_WORK_DIR",
		ApprovalMode: "",
		InstallHint:  "Install with: npm install -g @anthropic-ai/claude-code",
		JoinMessage:  "Claude CLI agent online. I can analyze codebases, generate code, review, and help with architecture using Anthropic's Claude. @mention me to get started.",
	},
	"copilot": {
		Type:         "copilot",
		Command:      "github-copilot-cli",
		ProviderName: "copilot-cli",
		ModelName:    "copilot-agent",
		DefaultName:  "Copilot",
		BaseArgs:     nil,
		Expertise: []string{
			"Code Generation", "Code Review",
			"Codebase Analysis", "General Development",
			"Shell Commands",
		},
		EnvVars:      nil,
		WorkDirEnv:   "COPILOT_WORK_DIR",
		ApprovalMode: "",
		InstallHint:  "Install with: npm install -g @githubnext/github-copilot-cli",
		JoinMessage:  "Copilot CLI agent online. I can help with code generation and review using GitHub Copilot. @mention me to get started.",
	},
}

// GetCLIAgentConfig returns the config for a given CLI agent type.
// Returns the config and true if found, zero value and false otherwise.
func GetCLIAgentConfig(cliType string) (CLIAgentConfig, bool) {
	cfg, ok := cliAgentRegistry[cliType]
	return cfg, ok
}

// ListCLIAgentTypes returns all registered CLI agent type names, sorted.
func ListCLIAgentTypes() []string {
	types := make([]string, 0, len(cliAgentRegistry))
	for k := range cliAgentRegistry {
		types = append(types, k)
	}
	sort.Strings(types)
	return types
}

// NewCLIAgentFromConfig creates a CLI-backed agent from a registry config.
func NewCLIAgentFromConfig(cfg CLIAgentConfig, name string, provider ai.AIProvider, hub HubClient) *Agent {
	return NewAgentWithProvider(
		protocol.AgentTypeCLI,
		name,
		cfg.Expertise,
		provider,
		hub,
		cfg.ProviderName,
		cfg.ModelName,
	)
}
