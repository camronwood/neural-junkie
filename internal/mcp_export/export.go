package mcp_export

import (
	"encoding/json"
	"fmt"
	"time"
)

// AgentExport represents a complete MCP export of an agent's knowledge
type AgentExport struct {
	Version      string        `json:"version"`
	Agent        AgentMetadata `json:"agent"`
	Resources    []MCPResource `json:"resources"`
	Prompts      []MCPPrompt   `json:"prompts"`
	SystemPrompt string        `json:"systemPrompt"`
	ExportedAt   time.Time     `json:"exportedAt"`
}

// AgentMetadata contains agent configuration and metadata
type AgentMetadata struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"` // "repo" or "helper"
	Expertise     []string `json:"expertise"`
	Description   string   `json:"description,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	Repository    string   `json:"repository,omitempty"`    // For repo agents
	KnowledgePath string   `json:"knowledgePath,omitempty"` // For helper agents
}

// MCPResource represents a knowledge resource in MCP format
type MCPResource struct {
	URI      string `json:"uri"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Content  string `json:"content"`
	Size     int64  `json:"size,omitempty"`
}

// MCPPrompt represents a pre-configured prompt template
type MCPPrompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []PromptArgument `json:"arguments"`
	Prompt      string           `json:"prompt"`
	Resources   []string         `json:"resources,omitempty"` // URIs of resources to include
}

// PromptArgument defines a parameter for a prompt template
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "string", "number", "boolean"
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// ExportMetadata contains information about an export
type ExportMetadata struct {
	ExportPath    string    `json:"exportPath"`
	FileSize      int64     `json:"fileSize"`
	ResourceCount int       `json:"resourceCount"`
	PromptCount   int       `json:"promptCount"`
	LastModified  time.Time `json:"lastModified"`
}

// ExportableAgent interface for agents that can be exported to MCP format
type ExportableAgent interface {
	ExportToMCP() (*AgentExport, error)
	GetExportMetadata() *ExportMetadata
}

// CreateRepoResource creates a resource for repository content
func CreateRepoResource(uri, name, mimeType, content string) MCPResource {
	return MCPResource{
		URI:      uri,
		Name:     name,
		MimeType: mimeType,
		Content:  content,
		Size:     int64(len(content)),
	}
}

// CreateHelperResource creates a resource for helper agent content
func CreateHelperResource(uri, name, mimeType, content string) MCPResource {
	return MCPResource{
		URI:      uri,
		Name:     name,
		MimeType: mimeType,
		Content:  content,
		Size:     int64(len(content)),
	}
}

// CreatePrompt creates a prompt template
func CreatePrompt(name, description, prompt string, resources []string) MCPPrompt {
	return MCPPrompt{
		Name:        name,
		Description: description,
		Arguments:   []PromptArgument{},
		Prompt:      prompt,
		Resources:   resources,
	}
}

// CreatePromptWithArgs creates a prompt template with arguments
func CreatePromptWithArgs(name, description, prompt string, args []PromptArgument, resources []string) MCPPrompt {
	return MCPPrompt{
		Name:        name,
		Description: description,
		Arguments:   args,
		Prompt:      prompt,
		Resources:   resources,
	}
}

// ValidateExport validates an agent export
func ValidateExport(export *AgentExport) error {
	if export.Version == "" {
		return fmt.Errorf("export version is required")
	}

	if export.Agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}

	if export.Agent.Type == "" {
		return fmt.Errorf("agent type is required")
	}

	if len(export.Resources) == 0 {
		return fmt.Errorf("at least one resource is required")
	}

	if export.SystemPrompt == "" {
		return fmt.Errorf("system prompt is required")
	}

	// Validate resource URIs are unique
	uriMap := make(map[string]bool)
	for _, resource := range export.Resources {
		if resource.URI == "" {
			return fmt.Errorf("resource URI cannot be empty")
		}
		if uriMap[resource.URI] {
			return fmt.Errorf("duplicate resource URI: %s", resource.URI)
		}
		uriMap[resource.URI] = true
	}

	// Validate prompt names are unique
	promptMap := make(map[string]bool)
	for _, prompt := range export.Prompts {
		if prompt.Name == "" {
			return fmt.Errorf("prompt name cannot be empty")
		}
		if promptMap[prompt.Name] {
			return fmt.Errorf("duplicate prompt name: %s", prompt.Name)
		}
		promptMap[prompt.Name] = true
	}

	return nil
}

// ToJSON converts an agent export to JSON
func (export *AgentExport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(export, "", "  ")
}

// FromJSON creates an agent export from JSON
func FromJSON(data []byte) (*AgentExport, error) {
	var export AgentExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to unmarshal export: %w", err)
	}

	if err := ValidateExport(&export); err != nil {
		return nil, fmt.Errorf("invalid export: %w", err)
	}

	return &export, nil
}

// GetResourceByURI finds a resource by its URI
func (export *AgentExport) GetResourceByURI(uri string) *MCPResource {
	for _, resource := range export.Resources {
		if resource.URI == uri {
			return &resource
		}
	}
	return nil
}

// GetPromptByName finds a prompt by its name
func (export *AgentExport) GetPromptByName(name string) *MCPPrompt {
	for _, prompt := range export.Prompts {
		if prompt.Name == name {
			return &prompt
		}
	}
	return nil
}

// GetResourcesByType returns resources filtered by MIME type
func (export *AgentExport) GetResourcesByType(mimeType string) []MCPResource {
	var filtered []MCPResource
	for _, resource := range export.Resources {
		if resource.MimeType == mimeType {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// GetTotalSize returns the total size of all resources
func (export *AgentExport) GetTotalSize() int64 {
	var total int64
	for _, resource := range export.Resources {
		total += resource.Size
	}
	return total
}

// GetResourceCount returns the number of resources
func (export *AgentExport) GetResourceCount() int {
	return len(export.Resources)
}

// GetPromptCount returns the number of prompts
func (export *AgentExport) GetPromptCount() int {
	return len(export.Prompts)
}
