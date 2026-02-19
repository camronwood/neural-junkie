package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HelperAgentStorage manages persistent storage for helper agents
type HelperAgentStorage struct {
	baseDir string
}

// NewHelperAgentStorage creates a new storage manager
func NewHelperAgentStorage() (*HelperAgentStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(home, ".neural-junkie", "helpers")

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create helpers directory: %w", err)
	}

	return &HelperAgentStorage{baseDir: baseDir}, nil
}

// SaveConfig saves a helper agent configuration
func (s *HelperAgentStorage) SaveConfig(name string, config *HelperAgentConfig) error {
	configPath := filepath.Join(s.baseDir, name, "config.json")

	// Create agent directory
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create agent directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadConfig loads a helper agent configuration
func (s *HelperAgentStorage) LoadConfig(name string) (*HelperAgentConfig, error) {
	// Resolve the name to actual directory
	resolvedName, err := s.ResolveAgentName(name)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(s.baseDir, resolvedName, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config HelperAgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// ResolveAgentName resolves a user-provided name to the actual directory name
func (s *HelperAgentStorage) ResolveAgentName(name string) (string, error) {
	// Get all available agent directories
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no helper agents found")
		}
		return "", fmt.Errorf("failed to read helpers directory: %w", err)
	}

	// First try exact match
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == name {
			// Verify config exists
			configPath := filepath.Join(s.baseDir, entry.Name(), "config.json")
			if _, err := os.Stat(configPath); err == nil {
				return entry.Name(), nil
			}
		}
	}

	// Try case-insensitive match
	for _, entry := range entries {
		if entry.IsDir() && strings.EqualFold(entry.Name(), name) {
			// Verify config exists
			configPath := filepath.Join(s.baseDir, entry.Name(), "config.json")
			if _, err := os.Stat(configPath); err == nil {
				return entry.Name(), nil
			}
		}
	}

	// Try normalized match (remove hyphens, spaces, underscores, lowercase)
	normalizedName := s.normalizeName(name)
	for _, entry := range entries {
		if entry.IsDir() {
			normalizedEntry := s.normalizeName(entry.Name())
			if normalizedEntry == normalizedName {
				// Verify config exists
				configPath := filepath.Join(s.baseDir, entry.Name(), "config.json")
				if _, err := os.Stat(configPath); err == nil {
					return entry.Name(), nil
				}
			}
		}
	}

	// Try matching against config names (for cases like "DayOneExpert" -> "day-one")
	for _, entry := range entries {
		if entry.IsDir() {
			configPath := filepath.Join(s.baseDir, entry.Name(), "config.json")
			if _, err := os.Stat(configPath); err == nil {
				// Load config to check the name field
				if config, err := s.loadConfigFromPath(configPath); err == nil {
					// Try exact match with config name
					if config.Name == name {
						return entry.Name(), nil
					}
					// Try case-insensitive match with config name
					if strings.EqualFold(config.Name, name) {
						return entry.Name(), nil
					}
					// Try normalized match with config name
					normalizedConfigName := s.normalizeName(config.Name)
					if normalizedConfigName == normalizedName {
						return entry.Name(), nil
					}
				}
			}
		}
	}

	// If no match found, list available agents
	available := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			configPath := filepath.Join(s.baseDir, entry.Name(), "config.json")
			if _, err := os.Stat(configPath); err == nil {
				available = append(available, entry.Name())
			}
		}
	}

	return "", fmt.Errorf("agent '%s' not found. Available agents: %v", name, available)
}

// normalizeName normalizes a name by removing hyphens, spaces, underscores and lowercasing
func (s *HelperAgentStorage) normalizeName(name string) string {
	// Remove hyphens, spaces, underscores
	normalized := strings.ReplaceAll(name, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	// Convert to lowercase
	return strings.ToLower(normalized)
}

// loadConfigFromPath loads a config from a specific path (helper method)
func (s *HelperAgentStorage) loadConfigFromPath(configPath string) (*HelperAgentConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config HelperAgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// DeleteConfig deletes a helper agent configuration
func (s *HelperAgentStorage) DeleteConfig(name string) error {
	agentDir := filepath.Join(s.baseDir, name)

	if err := os.RemoveAll(agentDir); err != nil {
		return fmt.Errorf("failed to delete agent directory: %w", err)
	}

	return nil
}

// ListConfigs lists all saved helper agent configurations
func (s *HelperAgentStorage) ListConfigs() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read helpers directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if config.json exists
			configPath := filepath.Join(s.baseDir, entry.Name(), "config.json")
			if _, err := os.Stat(configPath); err == nil {
				names = append(names, entry.Name())
			}
		}
	}

	return names, nil
}

// GetKnowledgePath returns the knowledge base path for an agent
func (s *HelperAgentStorage) GetKnowledgePath(name string) string {
	return filepath.Join(s.baseDir, name, "knowledge")
}

// EnsureKnowledgePath creates the knowledge directory if it doesn't exist
func (s *HelperAgentStorage) EnsureKnowledgePath(name string) error {
	knowledgePath := s.GetKnowledgePath(name)
	return os.MkdirAll(knowledgePath, 0755)
}

// CreateDefaultTemplates creates default helper agent templates
func (s *HelperAgentStorage) CreateDefaultTemplates() error {
	templates := DefaultHelperAgentConfigs()

	for name, config := range templates {
		// Check if already exists
		if _, err := s.LoadConfig(name); err == nil {
			continue // Skip if exists
		}

		// Create knowledge directory
		if err := s.EnsureKnowledgePath(name); err != nil {
			return fmt.Errorf("failed to create knowledge directory for %s: %w", name, err)
		}

		// Update knowledge path to use storage path
		config.KnowledgePath = s.GetKnowledgePath(name)

		// Save config
		if err := s.SaveConfig(name, config); err != nil {
			return fmt.Errorf("failed to save template %s: %w", name, err)
		}

		// Create example knowledge file
		if err := s.createExampleKnowledge(name); err != nil {
			return fmt.Errorf("failed to create example knowledge for %s: %w", name, err)
		}
	}

	return nil
}

// createExampleKnowledge creates example knowledge files for templates
func (s *HelperAgentStorage) createExampleKnowledge(name string) error {
	knowledgePath := s.GetKnowledgePath(name)

	var content string
	switch name {
	case "day-one":
		content = `# Day One Setup Guide

## Development Environment

### Required Tools
1. **Git** - Version control
2. **IDE/Editor** - VSCode, IntelliJ, or your preferred editor
3. **Language Runtime** - Go 1.21+, Node.js 18+, etc.
4. **Docker** - For running services locally
5. **Make** - Build automation tool

### Setup Steps
1. Clone the repository
2. Copy env.example to env.local
3. Run 'make install' to install dependencies
4. Run 'make start-all' to start services

## Common First Day Questions

### Where do I find X?
- Configuration: Check the env.local file
- Documentation: See the docs/ directory
- Examples: Check examples/ directory

### How do I run tests?
Run 'make test' from the project root

### Who do I ask for help?
- Technical questions: Ask in #engineering
- Process questions: Ask your onboarding buddy
- Access issues: Contact IT

## Tips for Your First Week
1. Set up your development environment (Day 1)
2. Read the architecture documentation (Day 1-2)
3. Pick up a "good first issue" (Day 2-3)
4. Pair program with a team member (Day 3-4)
5. Deploy your first change (Day 4-5)
`
	case "testing-expert":
		content = `# Testing Best Practices

## Testing Philosophy
- Write tests before fixing bugs
- Aim for meaningful tests, not just coverage
- Test behavior, not implementation

## Test Types
1. **Unit Tests** - Test individual functions/methods
2. **Integration Tests** - Test component interactions
3. **E2E Tests** - Test complete user workflows

## Common Patterns
- Arrange-Act-Assert (AAA)
- Given-When-Then
- Test fixtures and factories

## Best Practices
- One assertion per test (when possible)
- Clear test names that describe what's being tested
- Independent tests that can run in any order
- Fast tests that don't rely on external services
`
	case "docs-expert":
		content = `# Documentation Guidelines

## Documentation Types
1. **README** - Project overview and quick start
2. **API Docs** - Endpoint specifications
3. **Architecture Docs** - System design
4. **Code Comments** - Inline explanations

## Writing Good Documentation
- Start with "why" before "how"
- Include examples and use cases
- Keep it up to date with code changes
- Use diagrams for complex concepts
- Make it scannable with headings and lists

## Documentation Checklist
- [ ] Clear title and purpose
- [ ] Prerequisites listed
- [ ] Step-by-step instructions
- [ ] Examples provided
- [ ] Common issues documented
- [ ] Last updated date
`
	default:
		content = fmt.Sprintf("# %s Knowledge Base\n\nAdd your knowledge documents here.\n", name)
	}

	exampleFile := filepath.Join(knowledgePath, "overview.md")
	return os.WriteFile(exampleFile, []byte(content), 0644)
}

// ListConfigsWithMetadata returns all cached helper agents with metadata
func (s *HelperAgentStorage) ListConfigsWithMetadata() ([]map[string]interface{}, error) {
	// Get all config names
	configNames, err := s.ListConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	var cachedAgents []map[string]interface{}

	for _, name := range configNames {
		// Load config
		config, err := s.LoadConfig(name)
		if err != nil {
			// Skip if can't load config, but continue with other agents
			continue
		}

		// Calculate knowledge base size
		knowledgePath := s.GetKnowledgePath(name)
		cacheSize := int64(0)
		if entries, err := os.ReadDir(knowledgePath); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					if info, err := entry.Info(); err == nil {
						cacheSize += info.Size()
					}
				}
			}
		}

		// Use current time as last_used since we don't track this yet
		// TODO: Add last_used tracking to helper storage
		lastUsed := "2025-10-15T10:30:00Z" // Placeholder

		agent := map[string]interface{}{
			"type":       "helper",
			"name":       config.Name,
			"path":       knowledgePath,
			"last_used":  lastUsed,
			"cache_size": cacheSize,
			"metadata": map[string]interface{}{
				"description": config.Description,
				"expertise":   config.Expertise,
				"keywords":    config.Keywords,
			},
		}

		cachedAgents = append(cachedAgents, agent)
	}

	return cachedAgents, nil
}
