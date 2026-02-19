package filechange

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileChangeExecutor handles execution of approved file changes
type FileChangeExecutor struct {
	workspaceRoot string
	backupDir     string
}

// NewFileChangeExecutor creates a new file change executor
func NewFileChangeExecutor(workspaceRoot string) *FileChangeExecutor {
	backupDir := filepath.Join(workspaceRoot, ".neural-junkie", "backups")
	return &FileChangeExecutor{
		workspaceRoot: workspaceRoot,
		backupDir:     backupDir,
	}
}

// ExecuteFileChange executes a file change operation
func (fce *FileChangeExecutor) ExecuteFileChange(change *FileChange) error {
	// Validate the change before execution
	if err := fce.validateFileChange(change); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(fce.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Execute based on operation type
	switch change.Operation {
	case FileOperationCreate:
		return fce.executeCreate(change)
	case FileOperationEdit:
		return fce.executeEdit(change)
	case FileOperationDelete:
		return fce.executeDelete(change)
	case FileOperationMove:
		return fce.executeMove(change)
	default:
		return fmt.Errorf("unknown operation: %s", change.Operation)
	}
}

// executeCreate creates a new file
func (fce *FileChangeExecutor) executeCreate(change *FileChange) error {
	// Ensure directory exists
	dir := filepath.Dir(change.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(change.FilePath, []byte(change.NewContent), 0644); err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

// executeEdit modifies an existing file
func (fce *FileChangeExecutor) executeEdit(change *FileChange) error {
	// Check if file exists
	if _, err := os.Stat(change.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", change.FilePath)
	}

	// Create backup
	if err := fce.createBackup(change.FilePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write the new content
	if err := os.WriteFile(change.FilePath, []byte(change.NewContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// executeDelete removes a file
func (fce *FileChangeExecutor) executeDelete(change *FileChange) error {
	// Check if file exists
	if _, err := os.Stat(change.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", change.FilePath)
	}

	// Create backup before deletion
	if err := fce.createBackup(change.FilePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Delete the file
	if err := os.Remove(change.FilePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// executeMove moves/renames a file
func (fce *FileChangeExecutor) executeMove(change *FileChange) error {
	// Check if source file exists
	if _, err := os.Stat(change.OldPath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", change.OldPath)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(change.NewPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create backup of source file
	if err := fce.createBackup(change.OldPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Move the file
	if err := os.Rename(change.OldPath, change.NewPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// validateFileChange validates a file change before execution
func (fce *FileChangeExecutor) validateFileChange(change *FileChange) error {
	// Validate paths
	if err := fce.validatePath(change.FilePath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if change.IsMoveOperation() {
		if err := fce.validatePath(change.OldPath); err != nil {
			return fmt.Errorf("invalid old path: %w", err)
		}
		if err := fce.validatePath(change.NewPath); err != nil {
			return fmt.Errorf("invalid new path: %w", err)
		}
	}

	// Check file size limits
	if len(change.NewContent) > 1024*1024 { // 1MB limit
		return fmt.Errorf("file content too large: %d bytes (max 1MB)", len(change.NewContent))
	}

	return nil
}

// validatePath validates a file path for security
func (fce *FileChangeExecutor) validatePath(path string) error {
	// Check for directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("directory traversal not allowed: %s", path)
	}

	// Ensure path is within workspace
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absWorkspace, err := filepath.Abs(fce.workspaceRoot)
	if err != nil {
		return fmt.Errorf("invalid workspace root: %w", err)
	}

	// Check if path is within workspace
	if !strings.HasPrefix(absPath, absWorkspace) {
		return fmt.Errorf("path outside workspace: %s", path)
	}

	return nil
}

// createBackup creates a backup of a file before modification
func (fce *FileChangeExecutor) createBackup(filePath string) error {
	// Read the current file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s_%s_%s",
		filepath.Base(filePath),
		timestamp,
		strings.ReplaceAll(filepath.Dir(filePath), "/", "_"))

	backupPath := filepath.Join(fce.backupDir, backupName)

	// Write backup
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// SetWorkspaceRoot updates the workspace root and backup directory.
// This allows the executor to be reconfigured when a workspace is resolved.
func (fce *FileChangeExecutor) SetWorkspaceRoot(root string) {
	fce.workspaceRoot = root
	fce.backupDir = filepath.Join(root, ".neural-junkie", "backups")
}

// GetWorkspaceRoot returns the current workspace root path.
func (fce *FileChangeExecutor) GetWorkspaceRoot() string {
	return fce.workspaceRoot
}

// GetFileContent reads the current content of a file
func (fce *FileChangeExecutor) GetFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// GetFileDiff generates a diff between old and new content
func (fce *FileChangeExecutor) GetFileDiff(oldContent, newContent string) (string, error) {
	// Simple diff implementation - in production, you might want to use a proper diff library
	if oldContent == newContent {
		return "No changes", nil
	}

	// Basic line-by-line diff
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var diff strings.Builder
	diff.WriteString("--- Old content\n")
	diff.WriteString("+++ New content\n")

	// Simple diff logic (this is basic - for production use a proper diff library)
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		oldLine := ""
		newLine := ""

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			diff.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", i+1, i+1))
			if oldLine != "" {
				diff.WriteString(fmt.Sprintf("-%s\n", oldLine))
			}
			if newLine != "" {
				diff.WriteString(fmt.Sprintf("+%s\n", newLine))
			}
		}
	}

	return diff.String(), nil
}
