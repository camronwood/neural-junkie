package agent

import (
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// NewCursorCLIAgent creates a Cursor CLI-backed agent.
// The agent uses the Cursor CLI in headless mode to generate responses,
// giving it access to Cursor's codebase search, file operations, and tool use.
func NewCursorCLIAgent(name string, provider ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Code Generation", "Refactoring", "Code Review",
		"Codebase Analysis", "Bug Fixing", "Testing",
		"Full-Stack Development", "Architecture",
		"File Operations", "Shell Commands",
	}

	return NewAgentWithProvider(
		protocol.AgentTypeCLI,
		name,
		expertise,
		provider,
		hub,
		"cursor-cli",
		"cursor-agent",
	)
}

// NewGeminiCLIAgent creates a Gemini CLI-backed agent.
// The agent uses the Gemini CLI in headless mode (--yolo) to generate responses,
// giving it access to Gemini's code generation, file operations, and tool use.
func NewGeminiCLIAgent(name string, provider ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Code Generation", "Code Review", "Multimodal Analysis",
		"Codebase Analysis", "Architecture", "Refactoring",
		"Testing", "Documentation",
		"File Operations", "Shell Commands",
	}

	return NewAgentWithProvider(
		protocol.AgentTypeCLI,
		name,
		expertise,
		provider,
		hub,
		"gemini-cli",
		"gemini-agent",
	)
}

// NewCLIAgent creates a generic CLI-backed agent with a custom provider name and model.
// Use this for non-Cursor CLI agents (e.g. Claude CLI, Copilot CLI).
func NewCLIAgent(name string, providerName string, provider ai.AIProvider, hub HubClient, expertise []string) *Agent {
	if len(expertise) == 0 {
		expertise = []string{
			"Code Generation", "Code Review",
			"Codebase Analysis", "General Development",
		}
	}

	return NewAgentWithProvider(
		protocol.AgentTypeCLI,
		name,
		expertise,
		provider,
		hub,
		providerName,
		provider.GetModel(),
	)
}
