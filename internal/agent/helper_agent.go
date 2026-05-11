package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// HelperAgent is a custom expert agent with a knowledge base
type HelperAgent struct {
	*Agent
	KnowledgePath string
	Config        *HelperAgentConfig
	Knowledge     *KnowledgeBase
}

// HelperAgentConfig defines the configuration for a helper agent
type HelperAgentConfig struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Expertise     []string `json:"expertise"`
	Keywords      []string `json:"keywords"` // Additional keywords to trigger responses
	SystemPrompt  string   `json:"system_prompt"`
	KnowledgePath string   `json:"knowledge_path"` // Path to knowledge base directory
}

// KnowledgeBase represents the agent's knowledge base
type KnowledgeBase struct {
	Documents map[string]string // filename -> content
	Index     []string          // List of topics/headings
}

// NewHelperAgent creates a new helper agent with custom knowledge
func NewHelperAgent(config *HelperAgentConfig, ai ai.AIProvider, hub HubClient) (*HelperAgent, error) {
	// Create base agent
	baseAgent := NewAgent(protocol.AgentTypeHelper, config.Name, config.Expertise, ai, hub)

	helperAgent := &HelperAgent{
		Agent:         baseAgent,
		KnowledgePath: config.KnowledgePath,
		Config:        config,
		Knowledge:     &KnowledgeBase{Documents: make(map[string]string)},
	}

	// Load knowledge base if path is provided (don't fail creation if path doesn't exist)
	if config.KnowledgePath != "" {
		// Try to load knowledge, but don't fail creation if it doesn't exist
		helperAgent.LoadKnowledge() // Ignore error during creation
	}

	return helperAgent, nil
}

// LoadKnowledge loads the knowledge base from the configured path
func (h *HelperAgent) LoadKnowledge() error {
	if h.KnowledgePath == "" {
		// No knowledge path configured - this is OK, agent will work without knowledge base
		return nil
	}

	// Expand home directory
	if strings.HasPrefix(h.KnowledgePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		h.KnowledgePath = filepath.Join(home, h.KnowledgePath[2:])
	}

	// Check if path exists
	info, err := os.Stat(h.KnowledgePath)
	if err != nil {
		return fmt.Errorf("knowledge path does not exist: %w", err)
	}

	// Load single file or directory
	if info.IsDir() {
		return h.loadKnowledgeDirectory()
	}
	return h.loadKnowledgeFile(h.KnowledgePath)
}

// loadKnowledgeDirectory loads all markdown files from a directory
func (h *HelperAgent) loadKnowledgeDirectory() error {
	log.Printf("[%s] Loading knowledge base from directory: %s", h.Info.Name, h.KnowledgePath)

	err := filepath.Walk(h.KnowledgePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only load markdown files
		if !info.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".txt")) {
			if err := h.loadKnowledgeFile(path); err != nil {
				log.Printf("[%s] Warning: failed to load %s: %v", h.Info.Name, path, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[%s] Loaded %d knowledge documents", h.Info.Name, len(h.Knowledge.Documents))
	return nil
}

// loadKnowledgeFile loads a single knowledge file
func (h *HelperAgent) loadKnowledgeFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Store with relative filename as key
	relPath, relErr := filepath.Rel(h.KnowledgePath, path)
	if relErr != nil {
		return fmt.Errorf("knowledge path rel: %w", relErr)
	}
	h.Knowledge.Documents[relPath] = string(content)

	// Extract topics/headings for indexing
	h.extractTopics(string(content))

	return nil
}

// extractTopics extracts markdown headings for quick reference
func (h *HelperAgent) extractTopics(content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			// Extract heading text
			heading := strings.TrimLeft(trimmed, "# ")
			h.Knowledge.Index = append(h.Knowledge.Index, heading)
		}
	}
}

// buildPrompt overrides the base agent's prompt to include knowledge base
func (h *HelperAgent) buildPromptWithKnowledge(msg *protocol.Message) string {
	var prompt strings.Builder

	// Custom system prompt if configured
	if h.Config.SystemPrompt != "" {
		prompt.WriteString(h.Config.SystemPrompt)
		prompt.WriteString("\n\n")
	} else {
		prompt.WriteString(fmt.Sprintf("You are %s, a specialized helper agent.\n\n", h.Info.Name))
	}

	prompt.WriteString(fmt.Sprintf("Description: %s\n\n", h.Config.Description))
	prompt.WriteString(fmt.Sprintf("Your expertise: %s\n\n", strings.Join(h.Info.Expertise, ", ")))

	// Self-knowledge: honest identity
	prompt.WriteString("=== YOUR TECHNICAL IDENTITY ===\n")
	prompt.WriteString(fmt.Sprintf("You are powered by the %q model via the %q provider.\n", h.Info.AIModel, h.Info.AIProvider))
	prompt.WriteString("If a user asks what model or LLM you are running, answer honestly with this information.\n")
	prompt.WriteString("Do NOT fabricate or guess your model architecture. Only state what is listed above.\n\n")

	// Include knowledge base context
	if len(h.Knowledge.Documents) > 0 {
		prompt.WriteString("=== KNOWLEDGE BASE ===\n\n")
		prompt.WriteString("You have access to the following documentation:\n\n")

		// Find relevant knowledge documents based on message content
		relevantDocs := h.findRelevantDocs(msg.Content)

		if len(relevantDocs) > 0 {
			for _, doc := range relevantDocs {
				prompt.WriteString(fmt.Sprintf("--- %s ---\n", doc.Name))
				prompt.WriteString(doc.Content)
				prompt.WriteString("\n\n")
			}
		} else {
			// Include topics index so agent knows what's available
			prompt.WriteString("Available topics:\n")
			for _, topic := range h.Knowledge.Index {
				prompt.WriteString(fmt.Sprintf("- %s\n", topic))
			}
			prompt.WriteString("\n")
		}

		prompt.WriteString("=== END KNOWLEDGE BASE ===\n\n")
	}

	// Standard instructions
	prompt.WriteString("Your role is to:\n")
	prompt.WriteString("1. Help users with questions in your area of expertise\n")
	prompt.WriteString("2. Reference your knowledge base when answering\n")
	prompt.WriteString("3. Be friendly, patient, and encouraging\n")
	prompt.WriteString("4. Provide step-by-step guidance when appropriate\n")
	prompt.WriteString("5. Ask clarifying questions if needed\n\n")

	// Insert system/user separator
	prompt.WriteString(ai.SystemPromptSeparator)

	// Append workspace context if the user shared it
	AppendWorkspaceContext(&prompt, msg)

	prompt.WriteString(fmt.Sprintf("User question from %s:\n%s\n\n", msg.From.Name, msg.Content))

	// Adaptive response length
	prompt.WriteString(getResponseLengthGuidance(msg.Content))

	return prompt.String()
}

// findRelevantDocs finds knowledge documents relevant to the query
func (h *HelperAgent) findRelevantDocs(query string) []struct {
	Name    string
	Content string
} {
	var relevant []struct {
		Name    string
		Content string
	}

	queryLower := strings.ToLower(query)

	// Simple keyword matching - could be enhanced with semantic search
	for name, content := range h.Knowledge.Documents {
		contentLower := strings.ToLower(content)

		// Check if query keywords appear in the document
		words := strings.Fields(queryLower)
		matches := 0
		for _, word := range words {
			if len(word) > 3 && strings.Contains(contentLower, word) {
				matches++
			}
		}

		// Include if multiple keywords match
		if matches >= 2 {
			relevant = append(relevant, struct {
				Name    string
				Content string
			}{Name: name, Content: content})
		}
	}

	// Limit to top 3 most relevant documents
	if len(relevant) > 3 {
		relevant = relevant[:3]
	}

	return relevant
}

// generateResponse overrides the base agent's response generation to include knowledge
func (h *HelperAgent) GenerateResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	prompt := h.buildPromptWithKnowledge(msg)

	// Get recent conversation history for context
	history := h.Context.History[msg.Channel]
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	response, err := h.AI.GenerateResponse(ctx, prompt, historyToMessages(history))
	if err != nil {
		return "", err
	}

	return response, nil
}

// shouldRespond overrides to add custom keyword matching
func (h *HelperAgent) ShouldRespond(msg *protocol.Message) bool {
	// Use base agent logic first
	if h.Agent.shouldRespond(msg) {
		return true
	}

	// Check custom keywords
	content := strings.ToLower(msg.Content)
	for _, keyword := range h.Config.Keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// GetConfig returns the helper agent configuration
func (h *HelperAgent) GetConfig() *HelperAgentConfig {
	return h.Config
}

// SaveConfig saves the helper agent configuration to disk
func (h *HelperAgent) SaveConfig(configPath string) error {
	// This would save the configuration as JSON
	// Implementation depends on how you want to store configs
	return nil
}

// LoadHelperAgentFromConfig loads a helper agent from a configuration file
func LoadHelperAgentFromConfig(configPath string, ai ai.AIProvider, hub HubClient) (*HelperAgent, error) {
	// This would load configuration from JSON file
	// For now, return error - to be implemented
	return nil, fmt.Errorf("not yet implemented")
}

// DefaultHelperAgentConfigs returns pre-configured helper agent templates
func DefaultHelperAgentConfigs() map[string]*HelperAgentConfig {
	return map[string]*HelperAgentConfig{
		"day-one": {
			Name:        "DayOneExpert",
			Description: "Helps new engineers get set up with dev environment, tools, and onboarding tasks",
			Expertise: []string{
				"Onboarding",
				"Development Environment Setup",
				"Local Environment Configuration",
				"Tool Installation",
				"Team Processes",
				"First Week Tasks",
			},
			Keywords: []string{
				"setup",
				"install",
				"onboard",
				"getting started",
				"first day",
				"new engineer",
				"environment",
				"configure",
			},
			SystemPrompt: `You are the Day One Expert, a friendly and patient guide for new engineers.
Your mission is to make the first day/week as smooth as possible by helping with:
- Development environment setup
- Tool installations and configurations
- Understanding team processes and workflows
- Answering "where do I find X" questions
- Providing context about the codebase and architecture

Be encouraging, assume the person is new, and provide clear step-by-step instructions.`,
			KnowledgePath: "~/.neural-junkie/helpers/day-one/",
		},
		"testing-expert": {
			Name:        "TestingExpert",
			Description: "Guides developers on testing practices, frameworks, and strategies",
			Expertise: []string{
				"Unit Testing",
				"Integration Testing",
				"E2E Testing",
				"Test-Driven Development",
				"Mocking",
				"Test Coverage",
			},
			Keywords: []string{
				"test",
				"testing",
				"tdd",
				"mock",
				"coverage",
				"assertion",
			},
			SystemPrompt: `You are the Testing Expert, helping developers write better tests.
You provide guidance on testing strategies, frameworks, and best practices.
Be practical and provide code examples when helpful.`,
			KnowledgePath: "~/.neural-junkie/helpers/testing/",
		},
		"docs-expert": {
			Name:        "DocumentationExpert",
			Description: "Helps with documentation standards, API docs, and knowledge sharing",
			Expertise: []string{
				"Documentation",
				"API Documentation",
				"README Files",
				"Code Comments",
				"Knowledge Sharing",
			},
			Keywords: []string{
				"documentation",
				"docs",
				"readme",
				"document",
				"api docs",
			},
			SystemPrompt: `You are the Documentation Expert, passionate about clear and helpful docs.
You help developers write better documentation, READMEs, and API docs.
Emphasize clarity, examples, and keeping docs up to date.`,
			KnowledgePath: "~/.neural-junkie/helpers/docs/",
		},
	}
}
