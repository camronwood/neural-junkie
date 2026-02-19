package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp_export"
)

// MCP export functionality temporarily disabled due to compilation issues

// ExportToMCP exports the helper agent's knowledge to MCP format
func (h *HelperAgent) ExportToMCP() (*mcp_export.AgentExport, error) {
	// Create agent metadata
	metadata := mcp_export.AgentMetadata{
		Name:          h.Info.Name,
		Type:          "helper",
		Expertise:     h.Info.Expertise,
		Description:   h.Config.Description,
		Keywords:      h.Config.Keywords,
		CreatedAt:     time.Now().Format("2006-01-02T15:04:05Z"),
		KnowledgePath: h.KnowledgePath,
	}

	// Create resources from knowledge base
	resources := h.createResourcesFromKnowledge()

	// Create prompt templates
	prompts := h.createPromptTemplates()

	// Generate system prompt
	systemPrompt := h.generateSystemPrompt()

	// Create the export
	export := &mcp_export.AgentExport{
		Version:      "1.0",
		Agent:        metadata,
		Resources:    resources,
		Prompts:      prompts,
		SystemPrompt: systemPrompt,
		ExportedAt:   time.Now(),
	}

	return export, nil
}

// GetExportMetadata returns metadata about the export
func (h *HelperAgent) GetExportMetadata() *mcp_export.ExportMetadata {
	// Calculate resource count
	resourceCount := 1 + // config
		len(h.Knowledge.Documents) + // knowledge documents
		1 // expertise

	// Calculate prompt count (standard prompts)
	promptCount := 4 // ask_question, get_guidance, find_topic, suggest_resources

	return &mcp_export.ExportMetadata{
		ResourceCount: resourceCount,
		PromptCount:   promptCount,
	}
}

// createResourcesFromKnowledge creates MCP resources from the knowledge base
func (h *HelperAgent) createResourcesFromKnowledge() []mcp_export.MCPResource {
	var resources []mcp_export.MCPResource

	// Agent configuration
	configContent := h.formatConfig()
	configResource := mcp_export.CreateHelperResource(
		"helper://config",
		"Agent Configuration",
		"application/json",
		configContent,
	)
	resources = append(resources, configResource)

	// Knowledge documents
	for filename, content := range h.Knowledge.Documents {
		uri := fmt.Sprintf("helper://knowledge/%s", filename)
		mimeType := h.getMimeTypeForFile(filename)

		resource := mcp_export.CreateHelperResource(
			uri,
			fmt.Sprintf("Knowledge: %s", filename),
			mimeType,
			content,
		)
		resources = append(resources, resource)
	}

	// Expertise areas
	expertiseContent := strings.Join(h.Info.Expertise, "\n")
	expertiseResource := mcp_export.CreateHelperResource(
		"helper://expertise",
		"Expertise Areas",
		"text/plain",
		expertiseContent,
	)
	resources = append(resources, expertiseResource)

	// Keywords
	if len(h.Config.Keywords) > 0 {
		keywordsContent := strings.Join(h.Config.Keywords, "\n")
		keywordsResource := mcp_export.CreateHelperResource(
			"helper://keywords",
			"Keywords",
			"text/plain",
			keywordsContent,
		)
		resources = append(resources, keywordsResource)
	}

	// Knowledge index
	if len(h.Knowledge.Index) > 0 {
		indexContent := strings.Join(h.Knowledge.Index, "\n")
		indexResource := mcp_export.CreateHelperResource(
			"helper://index",
			"Knowledge Index",
			"text/plain",
			indexContent,
		)
		resources = append(resources, indexResource)
	}

	return resources
}

// createPromptTemplates creates MCP prompt templates
func (h *HelperAgent) createPromptTemplates() []mcp_export.MCPPrompt {
	var prompts []mcp_export.MCPPrompt

	// Ask question prompt
	askQuestionPrompt := mcp_export.CreatePromptWithArgs(
		"ask_question",
		"Ask a question to the helper agent",
		"Using your knowledge base, answer this question: {{question}}. Reference relevant documentation and provide helpful, step-by-step guidance.",
		[]mcp_export.PromptArgument{
			{Name: "question", Description: "The question to ask", Type: "string", Required: true},
		},
		[]string{"helper://expertise", "helper://index"},
	)
	prompts = append(prompts, askQuestionPrompt)

	// Get guidance prompt
	guidancePrompt := mcp_export.CreatePromptWithArgs(
		"get_guidance",
		"Get guidance on a specific topic",
		"Using your expertise in {{helper://expertise}} and knowledge base, provide comprehensive guidance on {{topic}}. Include relevant documentation and best practices.",
		[]mcp_export.PromptArgument{
			{Name: "topic", Description: "Topic to get guidance on", Type: "string", Required: true},
		},
		[]string{"helper://expertise", "helper://index"},
	)
	prompts = append(prompts, guidancePrompt)

	// Find topic prompt
	findTopicPrompt := mcp_export.CreatePromptWithArgs(
		"find_topic",
		"Find information about a specific topic",
		"Search through your knowledge base for information about {{topic}}. Provide relevant documentation and explain how it relates to your expertise areas.",
		[]mcp_export.PromptArgument{
			{Name: "topic", Description: "Topic to search for", Type: "string", Required: true},
		},
		[]string{"helper://index"},
	)
	prompts = append(prompts, findTopicPrompt)

	// Suggest resources prompt
	suggestResourcesPrompt := mcp_export.CreatePrompt(
		"suggest_resources",
		"Suggest relevant resources and documentation",
		"Based on your knowledge base and expertise, suggest relevant resources, documentation, and next steps for the user's current needs.",
		[]string{"helper://expertise", "helper://index"},
	)
	prompts = append(prompts, suggestResourcesPrompt)

	return prompts
}

// generateSystemPrompt creates the system prompt for the exported agent
func (h *HelperAgent) generateSystemPrompt() string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are %s, a specialized helper agent.\n\n", h.Info.Name))

	prompt.WriteString(fmt.Sprintf("Description: %s\n\n", h.Config.Description))

	prompt.WriteString(fmt.Sprintf("Your expertise: %s\n\n", strings.Join(h.Info.Expertise, ", ")))

	if len(h.Config.Keywords) > 0 {
		prompt.WriteString(fmt.Sprintf("Keywords: %s\n\n", strings.Join(h.Config.Keywords, ", ")))
	}

	// Include custom system prompt if configured
	if h.Config.SystemPrompt != "" {
		prompt.WriteString("=== CUSTOM SYSTEM PROMPT ===\n")
		prompt.WriteString(h.Config.SystemPrompt)
		prompt.WriteString("\n\n")
	}

	// Include knowledge base context
	if len(h.Knowledge.Documents) > 0 {
		prompt.WriteString("=== KNOWLEDGE BASE ===\n")
		prompt.WriteString("You have access to the following documentation:\n\n")

		// List available topics
		if len(h.Knowledge.Index) > 0 {
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

	prompt.WriteString("## Available Resources\n")
	prompt.WriteString("You have access to the following resources:\n")
	prompt.WriteString("- Agent configuration and expertise\n")
	prompt.WriteString("- Knowledge base documents\n")
	prompt.WriteString("- Topic index for quick reference\n")
	prompt.WriteString("- Keywords for matching relevant content\n\n")

	prompt.WriteString("## Response Guidelines\n")
	prompt.WriteString("- Provide helpful, concise responses (2-4 sentences unless more detail is needed)\n")
	prompt.WriteString("- Reference specific documentation when relevant\n")
	prompt.WriteString("- Be encouraging and supportive\n")
	prompt.WriteString("- Ask clarifying questions if the request is unclear\n")
	prompt.WriteString("- Suggest next steps or additional resources when appropriate\n")

	return prompt.String()
}

// Helper methods

func (h *HelperAgent) formatConfig() string {

	// Convert to JSON (simplified for this example)
	var result strings.Builder
	result.WriteString("{\n")
	result.WriteString(fmt.Sprintf("  \"name\": \"%s\",\n", h.Config.Name))
	result.WriteString(fmt.Sprintf("  \"description\": \"%s\",\n", h.Config.Description))
	result.WriteString("  \"expertise\": [\n")
	for i, exp := range h.Info.Expertise {
		if i > 0 {
			result.WriteString(",\n")
		}
		result.WriteString(fmt.Sprintf("    \"%s\"", exp))
	}
	result.WriteString("\n  ],\n")

	if len(h.Config.Keywords) > 0 {
		result.WriteString("  \"keywords\": [\n")
		for i, keyword := range h.Config.Keywords {
			if i > 0 {
				result.WriteString(",\n")
			}
			result.WriteString(fmt.Sprintf("    \"%s\"", keyword))
		}
		result.WriteString("\n  ],\n")
	}

	result.WriteString(fmt.Sprintf("  \"knowledgePath\": \"%s\"\n", h.KnowledgePath))
	result.WriteString("}")

	return result.String()
}

func (h *HelperAgent) getMimeTypeForFile(filename string) string {
	if strings.HasSuffix(filename, ".md") {
		return "text/markdown"
	} else if strings.HasSuffix(filename, ".txt") {
		return "text/plain"
	} else if strings.HasSuffix(filename, ".json") {
		return "application/json"
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		return "text/yaml"
	} else {
		return "text/plain"
	}
}
