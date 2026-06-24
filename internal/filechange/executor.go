package filechange

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/fileedit"
	"github.com/camronwood/neural-junkie/internal/pathutil"
)

// FileChangeExecutor handles execution of approved file changes
type FileChangeExecutor struct {
	workspaceRoot string
	backupDir     string
	workspaceIO   WorkspaceIO
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
	if fce.workspaceIO == nil {
		dir := filepath.Dir(change.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	content := SanitizeFileChangeContent(change.NewContent)
	if err := fce.ioWrite(change.FilePath, []byte(content)); err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

// executeEdit modifies an existing file
func (fce *FileChangeExecutor) executeEdit(change *FileChange) error {
	if _, err := fce.ioStat(change.FilePath); err != nil {
		return fmt.Errorf("file does not exist: %s", change.FilePath)
	}
	if err := fce.createBackup(change.FilePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	content := SanitizeFileChangeContent(change.NewContent)
	if err := fce.ioWrite(change.FilePath, []byte(content)); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// executeDelete removes a file
func (fce *FileChangeExecutor) executeDelete(change *FileChange) error {
	if _, err := fce.ioStat(change.FilePath); err != nil {
		return fmt.Errorf("file does not exist: %s", change.FilePath)
	}
	if err := fce.createBackup(change.FilePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	if err := fce.ioRemove(change.FilePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// executeMove moves/renames a file
func (fce *FileChangeExecutor) executeMove(change *FileChange) error {
	if _, err := fce.ioStat(change.OldPath); err != nil {
		return fmt.Errorf("source file does not exist: %s", change.OldPath)
	}
	if err := fce.createBackup(change.OldPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	if err := fce.ioRename(change.OldPath, change.NewPath); err != nil {
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
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(fce.workspaceRoot, candidate)
	}
	if _, err := pathutil.WithinRoot(fce.workspaceRoot, candidate); err != nil {
		return fmt.Errorf("path outside workspace: %w", err)
	}
	return nil
}

// createBackup creates a backup of a file before modification
func (fce *FileChangeExecutor) createBackup(filePath string) error {
	content, err := fce.ioRead(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}
	if fce.workspaceIO != nil {
		rel, err := fce.absToRel(filePath)
		if err != nil {
			return err
		}
		backupRel := filepath.ToSlash(filepath.Join(".neural-junkie", "backups", rel+"_"+filepath.Base(filePath)+"_backup"))
		return fce.workspaceIO.WriteFile(context.Background(), backupRel, content)
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

// SetWorkspaceIO configures remote/local backend IO (nil = local os.*).
func (fce *FileChangeExecutor) SetWorkspaceIO(io WorkspaceIO) {
	fce.workspaceIO = io
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
	content, err := fce.ioRead(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// GetFileDiff generates a unified diff between old and new content.
func (fce *FileChangeExecutor) GetFileDiff(oldContent, newContent string) (string, error) {
	if oldContent == newContent {
		return "No changes", nil
	}
	diff := fileedit.UnifiedDiff("file", oldContent, newContent)
	if diff == "" {
		return "No changes", nil
	}
	return diff, nil
}
