package agent

import (
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CLIAgentConfig describes a CLI tool that can be wrapped as an agent.
type CLIAgentConfig struct {
	Type              string              // Registry key: "cursor", "gemini", "aider", "opencode", etc.
	Command           string              // Primary binary on PATH
	AlternateCommands []string            // Other binary names tried if Command is missing
	AlternateBaseArgs map[string][]string // Per-binary base args when AlternateCommands is used
	ProviderName      string              // "cursor-cli", "gemini-cli", etc.
	ModelName         string              // "cursor-agent", "gemini-agent", etc.
	DefaultName       string              // Default agent display name
	BaseArgs          []string            // Default provider base args (before prompt)
	Expertise         []string            // Agent expertise list
	EnvVars           []string            // Env var names to forward (e.g. ["CURSOR_API_KEY"])
	WorkDirEnv        string              // Env var for work dir override (e.g. "CURSOR_WORK_DIR")
	ApprovalMode      string              // "interactive", "auto_edit", "yolo", or ""
	InstallHint       string              // Help text if binary not found
	Install           *CLIInstallSpec     // Structured one-click install metadata
	Auth              *CLIAuthSpec        // Structured auth / login metadata
	JoinMessage       string              // Message sent to channel on agent join
}

// cliDevExpertise is the default skill list for repo-oriented CLI agents.
var cliDevExpertise = []string{
	"Code Generation", "Code Review", "Architecture",
	"Codebase Analysis", "Refactoring", "Bug Fixing",
	"Testing", "Documentation",
	"File Operations", "Shell Commands",
}

var cliAgentRegistry = map[string]CLIAgentConfig{
	"aider": {
		Type:         "aider",
		Command:      "aider",
		ProviderName: "aider-cli",
		ModelName:    "aider",
		DefaultName:  "Aider",
		BaseArgs:     []string{"--yes", "--message"},
		Expertise:    cliDevExpertise,
		EnvVars:      []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "AIDER_MODEL"},
		WorkDirEnv:   "AIDER_WORK_DIR",
		InstallHint:  "Install with: pip install aider-install  OR  pip install aider-chat",
		JoinMessage:  "Aider CLI agent online. I can pair-program in your repo with git-aware edits. @mention me to get started.",
	},
	"amazonq": {
		Type:              "amazonq",
		Command:           "q",
		AlternateCommands: []string{"amazon-q"},
		ProviderName:      "amazonq-cli",
		ModelName:         "amazon-q-agent",
		DefaultName:       "Amazon Q",
		BaseArgs:          []string{"chat", "--no-interactive", "-y"},
		Expertise:         append([]string(nil), cliDevExpertise...),
		EnvVars:           nil,
		WorkDirEnv:        "AMAZONQ_WORK_DIR",
		InstallHint:       "Install AWS Amazon Q Developer CLI (binary: q). See https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/command-line.html",
		JoinMessage:       "Amazon Q CLI agent online. I can help with code and AWS-aware development tasks. @mention me to get started.",
	},
	"amp": {
		Type:         "amp",
		Command:      "amp",
		ProviderName: "amp-cli",
		ModelName:    "amp-agent",
		DefaultName:  "Amp",
		BaseArgs:     []string{"--execute"},
		Expertise:    cliDevExpertise,
		EnvVars:      []string{"AMP_API_KEY"},
		WorkDirEnv:   "AMP_WORK_DIR",
		InstallHint:  "Install with: curl -fsSL https://ampcode.com/install.sh | bash  OR  brew install ampcode/tap/ampcode",
		JoinMessage:  "Amp CLI agent online. I can help with codebase-aware tasks using Sourcegraph Amp. @mention me to get started.",
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
		Install: &CLIInstallSpec{
			Method:  "npm",
			Command: "npm install -g @anthropic-ai/claude-code",
			Prereqs: []string{"node", "npm"},
		},
		Auth: &CLIAuthSpec{
			Method:       "cli_login",
			EnvVars:      []string{"ANTHROPIC_API_KEY"},
			LoginCommand: []string{"claude", "login"},
			ProbeCommand: []string{"claude", "auth", "status"},
			CredentialPaths: []string{
				"~/.claude/.credentials.json",
			},
		},
		JoinMessage: "Claude CLI agent online. I can analyze codebases, generate code, review, and help with architecture using Anthropic's Claude. @mention me to get started.",
	},
	"codex": {
		Type:         "codex",
		Command:      "codex",
		ProviderName: "codex-cli",
		ModelName:    "codex-agent",
		DefaultName:  "Codex",
		BaseArgs:     []string{"exec"},
		Expertise:    cliDevExpertise,
		EnvVars:      nil,
		WorkDirEnv:   "CODEX_WORK_DIR",
		InstallHint:  "Install with: brew install codex  OR  see https://github.com/openai/codex",
		JoinMessage:  "Codex CLI agent online. I can analyze codebases, generate code, and run tasks using OpenAI Codex. @mention me to get started.",
	},
	"copilot": {
		Type:              "copilot",
		Command:           "copilot",
		AlternateCommands: []string{"github-copilot-cli"},
		AlternateBaseArgs: map[string][]string{
			"github-copilot-cli": nil,
		},
		ProviderName: "copilot-cli",
		ModelName:    "copilot-agent",
		DefaultName:  "Copilot",
		BaseArgs:     []string{"-p"},
		Expertise: []string{
			"Code Generation", "Code Review",
			"Codebase Analysis", "General Development",
			"Shell Commands",
		},
		EnvVars:      nil,
		WorkDirEnv:   "COPILOT_WORK_DIR",
		ApprovalMode: "",
		InstallHint:  "Install with: brew install copilot-cli  OR  npm install -g @github/copilot  (legacy: npm install -g @githubnext/github-copilot-cli)",
		JoinMessage:  "Copilot CLI agent online. I can help with code generation and review using GitHub Copilot. @mention me to get started.",
	},
	"crush": {
		Type:         "crush",
		Command:      "crush",
		ProviderName: "crush-cli",
		ModelName:    "crush-agent",
		DefaultName:  "Crush",
		BaseArgs:     []string{"run"},
		Expertise:    cliDevExpertise,
		EnvVars:      nil,
		WorkDirEnv:   "CRUSH_WORK_DIR",
		InstallHint:  "Install with: brew install charmbracelet/tap/crush  OR  see https://github.com/charmbracelet/crush",
		JoinMessage:  "Crush CLI agent online. I can run coding tasks using Charm Crush. @mention me to get started.",
	},
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
		ApprovalMode: "yolo",
		InstallHint:  "Install with: curl https://cursor.com/install -fsS | bash",
		Install: &CLIInstallSpec{
			Method:  "curl",
			Command: "curl https://cursor.com/install -fsS | bash",
			Prereqs: []string{"curl", "bash"},
		},
		Auth: &CLIAuthSpec{
			Method:       "cli_login",
			EnvVars:      []string{"CURSOR_API_KEY"},
			LoginCommand: []string{"agent", "login"},
			ProbeCommand: []string{"agent", "status"},
		},
		JoinMessage: "Cursor CLI agent online. I can analyze codebases, generate code, refactor, and run shell commands using Cursor's agent capabilities. @mention me to get started.",
	},
	"droid": {
		Type:         "droid",
		Command:      "droid",
		ProviderName: "droid-cli",
		ModelName:    "droid-agent",
		DefaultName:  "Droid",
		BaseArgs:     []string{"exec", "--auto", "high"},
		Expertise:    cliDevExpertise,
		EnvVars:      nil,
		WorkDirEnv:   "DROID_WORK_DIR",
		InstallHint:  "Install Factory CLI (binary: droid). See https://docs.factory.ai/",
		JoinMessage:  "Factory Droid CLI agent online. I can run headless coding tasks with droid exec. @mention me to get started.",
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
		Install: &CLIInstallSpec{
			Method:  "npm",
			Command: "npm install -g @google/gemini-cli",
			Prereqs: []string{"node", "npm"},
		},
		Auth: &CLIAuthSpec{
			Method:       "cli_login",
			LoginCommand: []string{"gemini", "auth", "login"},
			ProbeCommand: []string{"gemini", "auth", "status"},
			CredentialPaths: []string{
				"~/.gemini/oauth_creds.json",
			},
		},
		JoinMessage: "Gemini CLI agent online. I can analyze codebases, generate code, review, and run shell commands using Google's Gemini agent. @mention me to get started.",
	},
	"kiro": {
		Type:              "kiro",
		Command:           "kiro-cli",
		AlternateCommands: []string{"kiro"},
		ProviderName:      "kiro-cli",
		ModelName:         "kiro-agent",
		DefaultName:       "Kiro",
		BaseArgs:          []string{"chat", "--no-interactive"},
		Expertise:         append([]string(nil), cliDevExpertise...),
		EnvVars:           nil,
		WorkDirEnv:        "KIRO_WORK_DIR",
		InstallHint:       "Install Kiro CLI (kiro-cli). See https://kiro.dev/docs/cli/",
		JoinMessage:       "Kiro CLI agent online. I can help with code tasks using Kiro. @mention me to get started.",
	},
	"opencode": {
		Type:         "opencode",
		Command:      "opencode",
		ProviderName: "opencode-cli",
		ModelName:    "opencode-agent",
		DefaultName:  "OpenCode",
		BaseArgs:     []string{"-p", "-q"},
		Expertise:    cliDevExpertise,
		EnvVars:      nil,
		WorkDirEnv:   "OPENCODE_WORK_DIR",
		InstallHint:  "Install with: npm install -g opencode-ai  OR  see https://opencode.ai/docs/cli/",
		JoinMessage:  "OpenCode CLI agent online. I can run non-interactive prompts against your codebase. @mention me to get started.",
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
	model := cfg.ModelName
	if provider != nil {
		if m := strings.TrimSpace(provider.GetModel()); m != "" {
			model = m
		}
	}
	return NewAgentWithProvider(
		protocol.AgentTypeCLI,
		name,
		cfg.Expertise,
		provider,
		hub,
		cfg.ProviderName,
		model,
	)
}
