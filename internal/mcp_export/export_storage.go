package mcp_export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ExportsDir = ".neural-junkie/exports"
	RepoDir    = "repo"
	HelperDir  = "helper"
)

// ExportStorage manages exported agent packages
type ExportStorage struct {
	baseDir string
}

// resolveExportsBaseDir returns the exports directory from MCP_EXPORTS_DIR or ~/.neural-junkie/exports.
func resolveExportsBaseDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("MCP_EXPORTS_DIR")); dir != "" {
		if strings.HasPrefix(dir, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			return filepath.Join(homeDir, dir[2:]), nil
		}
		return filepath.Clean(dir), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ExportsDir), nil
}

// NewExportStorage creates a new export storage manager
func NewExportStorage() (*ExportStorage, error) {
	baseDir, err := resolveExportsBaseDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create exports directory: %w", err)
	}

	// Create subdirectories
	if err := os.MkdirAll(filepath.Join(baseDir, RepoDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create repo exports directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(baseDir, HelperDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create helper exports directory: %w", err)
	}

	return &ExportStorage{baseDir: baseDir}, nil
}

// SaveExport saves an agent export to disk
func (s *ExportStorage) SaveExport(export *AgentExport) error {
	if err := ValidateExport(export); err != nil {
		return fmt.Errorf("invalid export: %w", err)
	}

	// Determine subdirectory based on agent type
	var subDir string
	switch export.Agent.Type {
	case "repo":
		subDir = RepoDir
	case "helper":
		subDir = HelperDir
	default:
		return fmt.Errorf("unsupported agent type: %s", export.Agent.Type)
	}

	// Generate filename from agent name
	filename := s.sanitizeFilename(export.Agent.Name) + ".json"
	exportPath := filepath.Join(s.baseDir, subDir, filename)

	// Convert to JSON
	data, err := export.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}

	// Write to file
	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// LoadExport loads an agent export from disk
func (s *ExportStorage) LoadExport(agentName, agentType string) (*AgentExport, error) {
	// Determine subdirectory based on agent type
	var subDir string
	switch agentType {
	case "repo":
		subDir = RepoDir
	case "helper":
		subDir = HelperDir
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}

	// Generate filename from agent name
	filename := s.sanitizeFilename(agentName) + ".json"
	exportPath := filepath.Join(s.baseDir, subDir, filename)

	// Read file
	data, err := os.ReadFile(exportPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("export not found: %s", agentName)
		}
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	// Parse JSON
	export, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export: %w", err)
	}

	return export, nil
}

// LoadExportFromPath loads an agent export from a specific file path
func (s *ExportStorage) LoadExportFromPath(exportPath string) (*AgentExport, error) {
	// Read file
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	// Parse JSON
	export, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse export: %w", err)
	}

	return export, nil
}

// DeleteExport removes an agent export from disk
func (s *ExportStorage) DeleteExport(agentName, agentType string) error {
	// Determine subdirectory based on agent type
	var subDir string
	switch agentType {
	case "repo":
		subDir = RepoDir
	case "helper":
		subDir = HelperDir
	default:
		return fmt.Errorf("unsupported agent type: %s", agentType)
	}

	// Generate filename from agent name
	filename := s.sanitizeFilename(agentName) + ".json"
	exportPath := filepath.Join(s.baseDir, subDir, filename)

	// Check if file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		return fmt.Errorf("export not found: %s", agentName)
	}

	// Delete file
	if err := os.Remove(exportPath); err != nil {
		return fmt.Errorf("failed to delete export: %w", err)
	}

	return nil
}

// ListExports returns all available exports
func (s *ExportStorage) ListExports() ([]ExportInfo, error) {
	var exports []ExportInfo

	// List repo exports
	repoExports, err := s.listExportsInDir(RepoDir, "repo")
	if err != nil {
		return nil, fmt.Errorf("failed to list repo exports: %w", err)
	}
	exports = append(exports, repoExports...)

	// List helper exports
	helperExports, err := s.listExportsInDir(HelperDir, "helper")
	if err != nil {
		return nil, fmt.Errorf("failed to list helper exports: %w", err)
	}
	exports = append(exports, helperExports...)

	return exports, nil
}

// ExportInfo contains metadata about an export
type ExportInfo struct {
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	ExportPath    string    `json:"exportPath"`
	FileSize      int64     `json:"fileSize"`
	ResourceCount int       `json:"resourceCount"`
	PromptCount   int       `json:"promptCount"`
	LastModified  time.Time `json:"lastModified"`
	Description   string    `json:"description,omitempty"`
}

// listExportsInDir lists exports in a specific subdirectory
func (s *ExportStorage) listExportsInDir(subDir, agentType string) ([]ExportInfo, error) {
	dirPath := filepath.Join(s.baseDir, subDir)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExportInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var exports []ExportInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		exportPath := filepath.Join(dirPath, entry.Name())

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Try to load export to get metadata
		export, err := s.LoadExportFromPath(exportPath)
		if err != nil {
			// If we can't load the export, create basic info
			exportInfo := ExportInfo{
				Name:         strings.TrimSuffix(entry.Name(), ".json"),
				Type:         agentType,
				ExportPath:   exportPath,
				FileSize:     info.Size(),
				LastModified: info.ModTime(),
			}
			exports = append(exports, exportInfo)
			continue
		}

		// Create detailed info from export
		exportInfo := ExportInfo{
			Name:          export.Agent.Name,
			Type:          export.Agent.Type,
			ExportPath:    exportPath,
			FileSize:      export.GetTotalSize(),
			ResourceCount: export.GetResourceCount(),
			PromptCount:   export.GetPromptCount(),
			LastModified:  export.ExportedAt,
			Description:   export.Agent.Description,
		}
		exports = append(exports, exportInfo)
	}

	return exports, nil
}

// GetExportPath returns the file path for an export
func (s *ExportStorage) GetExportPath(agentName, agentType string) string {
	// Determine subdirectory based on agent type
	var subDir string
	switch agentType {
	case "repo":
		subDir = RepoDir
	case "helper":
		subDir = HelperDir
	default:
		return ""
	}

	// Generate filename from agent name
	filename := s.sanitizeFilename(agentName) + ".json"
	return filepath.Join(s.baseDir, subDir, filename)
}

// ExportExists checks if an export exists
func (s *ExportStorage) ExportExists(agentName, agentType string) bool {
	exportPath := s.GetExportPath(agentName, agentType)
	_, err := os.Stat(exportPath)
	return err == nil
}

// GetExportStats returns statistics about all exports
func (s *ExportStorage) GetExportStats() (*ExportStats, error) {
	exports, err := s.ListExports()
	if err != nil {
		return nil, fmt.Errorf("failed to list exports: %w", err)
	}

	stats := &ExportStats{
		TotalExports:   len(exports),
		RepoExports:    0,
		HelperExports:  0,
		TotalSize:      0,
		TotalResources: 0,
		TotalPrompts:   0,
		LastExport:     time.Time{},
	}

	for _, export := range exports {
		stats.TotalSize += export.FileSize
		stats.TotalResources += export.ResourceCount
		stats.TotalPrompts += export.PromptCount

		if export.Type == "repo" {
			stats.RepoExports++
		} else if export.Type == "helper" {
			stats.HelperExports++
		}

		if export.LastModified.After(stats.LastExport) {
			stats.LastExport = export.LastModified
		}
	}

	return stats, nil
}

// ExportStats contains statistics about exports
type ExportStats struct {
	TotalExports   int       `json:"totalExports"`
	RepoExports    int       `json:"repoExports"`
	HelperExports  int       `json:"helperExports"`
	TotalSize      int64     `json:"totalSize"`
	TotalResources int       `json:"totalResources"`
	TotalPrompts   int       `json:"totalPrompts"`
	LastExport     time.Time `json:"lastExport"`
}

// sanitizeFilename creates a safe filename from an agent name
func (s *ExportStorage) sanitizeFilename(name string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)

	sanitized := replacer.Replace(name)

	// Remove any remaining problematic characters
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	return result.String()
}

// CleanupOldExports removes exports older than the specified duration
func (s *ExportStorage) CleanupOldExports(olderThan time.Duration) (int, error) {
	exports, err := s.ListExports()
	if err != nil {
		return 0, fmt.Errorf("failed to list exports: %w", err)
	}

	cutoff := time.Now().Add(-olderThan)
	deletedCount := 0

	for _, export := range exports {
		if export.LastModified.Before(cutoff) {
			// Extract agent name from path
			agentName := strings.TrimSuffix(filepath.Base(export.ExportPath), ".json")
			if err := s.DeleteExport(agentName, export.Type); err != nil {
				// Log error but continue with other exports
				continue
			}
			deletedCount++
		}
	}

	return deletedCount, nil
}
