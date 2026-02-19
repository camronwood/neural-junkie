package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/repo"
)

// ExportToMCP exports the repo agent's knowledge to MCP format
func (ra *RepoAgent) ExportToMCP() (*mcp_export.AgentExport, error) {
	if ra.index == nil {
		return nil, fmt.Errorf("repository not indexed yet")
	}

	// Create agent metadata
	metadata := mcp_export.AgentMetadata{
		Name:        ra.Info.Name,
		Type:        "repo",
		Expertise:   ra.Info.Expertise,
		Description: fmt.Sprintf("Repository expert for %s", ra.index.Name),
		CreatedAt:   time.Now().Format(time.RFC3339),
		Repository:  ra.repoPath,
	}

	// Create resources from repository index
	resources := ra.createResourcesFromIndex()

	// Create prompt templates
	prompts := ra.createPromptTemplates()

	// Generate system prompt
	systemPrompt := ra.generateSystemPrompt()

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
func (ra *RepoAgent) GetExportMetadata() *mcp_export.ExportMetadata {
	if ra.index == nil {
		return &mcp_export.ExportMetadata{
			ResourceCount: 0,
			PromptCount:   0,
		}
	}

	// Calculate resource count
	resourceCount := 1 + // architecture
		len(ra.index.KeyFiles) + // key files
		len(ra.index.SourceFiles) + // source files
		1 // patterns

	// Calculate prompt count (standard prompts)
	promptCount := 5 // analyze_architecture, explain_code, find_files, get_dependencies, suggest_improvements

	return &mcp_export.ExportMetadata{
		ResourceCount: resourceCount,
		PromptCount:   promptCount,
	}
}

// createResourcesFromIndex creates MCP resources from the repository index
func (ra *RepoAgent) createResourcesFromIndex() []mcp_export.MCPResource {
	var resources []mcp_export.MCPResource

	// Architecture overview
	if ra.index.ArchitectureDoc != "" {
		resource := mcp_export.CreateRepoResource(
			"repo://architecture",
			"Architecture Overview",
			"text/markdown",
			ra.index.ArchitectureDoc,
		)
		resources = append(resources, resource)
	}

	// Key files (README, package.json, etc.)
	for filename, content := range ra.index.KeyFiles {
		uri := fmt.Sprintf("repo://files/%s", filename)
		mimeType := ra.getMimeTypeForFile(filename)

		resource := mcp_export.CreateRepoResource(
			uri,
			fmt.Sprintf("Key File: %s", filename),
			mimeType,
			content,
		)
		resources = append(resources, resource)
	}

	// Source files (limited to important ones)
	importantFiles := ra.getImportantSourceFiles()
	for _, file := range importantFiles {
		// Decompress content
		content, err := repo.DecompressContent(file.Content)
		if err != nil {
			continue // Skip files that can't be decompressed
		}

		uri := fmt.Sprintf("repo://files/%s", file.Path)
		mimeType := ra.getMimeTypeForLanguage(file.Language)

		resource := mcp_export.CreateRepoResource(
			uri,
			fmt.Sprintf("Source File: %s", file.Path),
			mimeType,
			content,
		)
		resources = append(resources, resource)
	}

	// Code patterns
	if len(ra.index.CodePatterns) > 0 {
		patternsContent := strings.Join(ra.index.CodePatterns, "\n")
		resource := mcp_export.CreateRepoResource(
			"repo://patterns",
			"Code Patterns",
			"text/plain",
			patternsContent,
		)
		resources = append(resources, resource)
	}

	// Dependencies
	if len(ra.index.Dependencies) > 0 {
		depsContent := ra.formatDependencies()
		resource := mcp_export.CreateRepoResource(
			"repo://dependencies",
			"Dependencies",
			"text/plain",
			depsContent,
		)
		resources = append(resources, resource)
	}

	// Git information
	if ra.index.GitInfo != nil {
		gitContent := ra.formatGitInfo()
		resource := mcp_export.CreateRepoResource(
			"repo://git",
			"Git Information",
			"text/plain",
			gitContent,
		)
		resources = append(resources, resource)
	}

	return resources
}

// createPromptTemplates creates MCP prompt templates
func (ra *RepoAgent) createPromptTemplates() []mcp_export.MCPPrompt {
	var prompts []mcp_export.MCPPrompt

	// Analyze architecture prompt
	analyzeArchPrompt := mcp_export.CreatePromptWithArgs(
		"analyze_architecture",
		"Analyze the architecture of the repository",
		"Using the architecture documentation at {{repo://architecture}}, analyze the overall architecture of {{repository_name}}. Explain the main components, their relationships, and how they work together.",
		[]mcp_export.PromptArgument{},
		[]string{"repo://architecture"},
	)
	prompts = append(prompts, analyzeArchPrompt)

	// Explain code prompt
	explainCodePrompt := mcp_export.CreatePromptWithArgs(
		"explain_code",
		"Explain specific code functionality",
		"Using the source file at {{repo://files/{{file_path}}}}, explain what this code does, its purpose, and how it fits into the overall system.",
		[]mcp_export.PromptArgument{
			{Name: "file_path", Description: "Path to the file to explain", Type: "string", Required: true},
		},
		[]string{"repo://files/{{file_path}}"},
	)
	prompts = append(prompts, explainCodePrompt)

	// Find files prompt
	findFilesPrompt := mcp_export.CreatePromptWithArgs(
		"find_files",
		"Find files related to a specific topic",
		"Based on the repository structure and code patterns at {{repo://patterns}}, find files related to {{topic}} and explain their purpose.",
		[]mcp_export.PromptArgument{
			{Name: "topic", Description: "Topic or functionality to search for", Type: "string", Required: true},
		},
		[]string{"repo://patterns"},
	)
	prompts = append(prompts, findFilesPrompt)

	// Get dependencies prompt
	dependenciesPrompt := mcp_export.CreatePrompt(
		"get_dependencies",
		"Get information about project dependencies",
		"Using the dependency information at {{repo://dependencies}}, explain the project's dependencies, their purposes, and any potential issues or updates needed.",
		[]string{"repo://dependencies"},
	)
	prompts = append(prompts, dependenciesPrompt)

	// Suggest improvements prompt
	improvementsPrompt := mcp_export.CreatePromptWithArgs(
		"suggest_improvements",
		"Suggest improvements for the codebase",
		"Based on the architecture at {{repo://architecture}} and code patterns at {{repo://patterns}}, suggest improvements for {{area}} the codebase. Consider performance, maintainability, and best practices.",
		[]mcp_export.PromptArgument{
			{Name: "area", Description: "Specific area to focus on (optional)", Type: "string", Required: false},
		},
		[]string{"repo://architecture", "repo://patterns"},
	)
	prompts = append(prompts, improvementsPrompt)

	return prompts
}

// generateSystemPrompt creates the system prompt for the exported agent
func (ra *RepoAgent) generateSystemPrompt() string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are %s, a repository expert agent with deep knowledge of the %s codebase.\n\n",
		ra.Info.Name, ra.index.Name))

	prompt.WriteString("## Repository Overview\n")
	prompt.WriteString(ra.index.ArchitectureDoc)
	prompt.WriteString("\n\n")

	prompt.WriteString("## Your Expertise\n")
	prompt.WriteString(fmt.Sprintf("You are an expert in: %s\n\n", strings.Join(ra.Info.Expertise, ", ")))

	prompt.WriteString("## Available Resources\n")
	prompt.WriteString("You have access to the following resources:\n")
	prompt.WriteString("- Architecture documentation\n")
	prompt.WriteString("- Source code files\n")
	prompt.WriteString("- Key configuration files\n")
	prompt.WriteString("- Code patterns and frameworks\n")
	prompt.WriteString("- Dependency information\n")
	prompt.WriteString("- Git history and recent changes\n\n")

	prompt.WriteString("## Your Role\n")
	prompt.WriteString("As a repository expert, you should:\n")
	prompt.WriteString("1. Answer questions about the code structure, architecture, and implementation\n")
	prompt.WriteString("2. Reference specific files and code snippets when relevant\n")
	prompt.WriteString("3. Explain how different parts of the codebase work together\n")
	prompt.WriteString("4. Provide concrete code examples from the repository\n")
	prompt.WriteString("5. Be specific and cite actual files/code/line numbers when possible\n")
	prompt.WriteString("6. If you reference code, quote the relevant parts directly\n\n")

	prompt.WriteString("## Repository Information\n")
	prompt.WriteString(fmt.Sprintf("- Repository: %s\n", ra.index.Name))
	prompt.WriteString(fmt.Sprintf("- Path: %s\n", ra.repoPath))
	prompt.WriteString(fmt.Sprintf("- Files: %d\n", ra.index.FileCount))
	prompt.WriteString(fmt.Sprintf("- Size: %d bytes\n", ra.index.TotalSize))
	prompt.WriteString(fmt.Sprintf("- Languages: %s\n", strings.Join(ra.getLanguages(), ", ")))

	return prompt.String()
}

// Helper methods

func (ra *RepoAgent) getImportantSourceFiles() []*repo.SourceFile {
	// Limit to most important files to avoid overwhelming the export
	var important []*repo.SourceFile

	// Add main entry points
	for _, file := range ra.index.SourceFiles {
		if ra.isImportantFile(file.Path) {
			important = append(important, file)
			if len(important) >= 10 { // Limit to 10 most important files
				break
			}
		}
	}

	return important
}

func (ra *RepoAgent) isImportantFile(path string) bool {
	// Check for common important file patterns
	importantPatterns := []string{
		"main.go", "main.py", "main.js", "main.ts",
		"index.js", "index.ts", "app.js", "app.ts",
		"server.go", "server.py", "server.js",
		"handler", "controller", "service", "router",
		"config", "settings", "constants",
	}

	baseName := strings.ToLower(filepath.Base(path))
	for _, pattern := range importantPatterns {
		if strings.Contains(baseName, pattern) {
			return true
		}
	}

	// Check file size (prefer smaller files for export)
	if file, exists := ra.index.SourceFiles[path]; exists {
		return file.Size < 50000 // Less than 50KB
	}

	return false
}

func (ra *RepoAgent) getMimeTypeForFile(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".toml":
		return "text/toml"
	case ".xml":
		return "text/xml"
	case ".txt":
		return "text/plain"
	default:
		return "text/plain"
	}
}

func (ra *RepoAgent) getMimeTypeForLanguage(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "text/x-go"
	case "javascript":
		return "text/javascript"
	case "typescript":
		return "text/typescript"
	case "python":
		return "text/x-python"
	case "java":
		return "text/x-java"
	case "rust":
		return "text/x-rust"
	case "c":
		return "text/x-c"
	case "cpp", "c++":
		return "text/x-c++"
	case "html":
		return "text/html"
	case "css":
		return "text/css"
	case "sql":
		return "text/x-sql"
	case "yaml", "yml":
		return "text/yaml"
	case "json":
		return "application/json"
	default:
		return "text/plain"
	}
}

func (ra *RepoAgent) formatDependencies() string {
	var result strings.Builder

	for manager, deps := range ra.index.Dependencies {
		result.WriteString(fmt.Sprintf("=== %s Dependencies ===\n", strings.ToUpper(manager)))
		for _, dep := range deps {
			result.WriteString(fmt.Sprintf("- %s\n", dep))
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (ra *RepoAgent) formatGitInfo() string {
	if ra.index.GitInfo == nil {
		return "No git information available"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Branch: %s\n", ra.index.GitInfo.Branch))
	result.WriteString(fmt.Sprintf("Last Commit: %s\n", ra.index.GitInfo.LastCommit))
	result.WriteString(fmt.Sprintf("Last Commit Message: %s\n", ra.index.GitInfo.LastCommitMsg))
	result.WriteString(fmt.Sprintf("Last Commit Date: %s\n\n", ra.index.GitInfo.LastCommitDate.Format("2006-01-02 15:04:05")))

	if len(ra.index.GitInfo.RecentCommits) > 0 {
		result.WriteString("Recent Commits:\n")
		for _, commit := range ra.index.GitInfo.RecentCommits {
			result.WriteString(fmt.Sprintf("- %s: %s (%s)\n",
				commit.Hash[:8], commit.Message, commit.Date.Format("2006-01-02")))
		}
	}

	return result.String()
}

func (ra *RepoAgent) getLanguages() []string {
	languageMap := make(map[string]bool)

	// Get languages from source files
	for _, file := range ra.index.SourceFiles {
		if file.Language != "" {
			languageMap[file.Language] = true
		}
	}

	// Convert to slice
	var languages []string
	for lang := range languageMap {
		languages = append(languages, lang)
	}

	return languages
}
