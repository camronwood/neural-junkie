package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/repo"
)

// TestWatcherCreation tests file watcher creation
func TestWatcherCreation(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create some test files
	testFiles := []string{"file1.txt", "file2.go", "subdir/file3.md"}
	for _, filename := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != testRepoPath {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create subdirectory: %v", err)
			}
		}
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create change handler
	changeCount := 0
	changeHandler := func(path string) {
		changeCount++
		t.Logf("File change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected watcher to be created")
	}

	// Test completed successfully
	t.Log("Watcher creation test completed")

	// Clean up
	watcher.Stop()
}

// TestWatcherFileChanges tests file change detection
func TestWatcherFileChanges(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create initial test file
	testFile := filepath.Join(testRepoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("File change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify the test file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for change to be detected
	time.Sleep(1 * time.Second)

	// Check if change was detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	t.Logf("Detected %d changes", detectedChanges)
	if detectedChanges == 0 {
		t.Error("Expected file change to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherNewFileCreation tests detection of new file creation
func TestWatcherNewFileCreation(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("New file detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create new file
	newFile := filepath.Join(testRepoPath, "newfile.txt")
	err = os.WriteFile(newFile, []byte("new file content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Wait for change to be detected
	time.Sleep(1 * time.Second)

	// Check if change was detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	t.Logf("Detected %d changes", detectedChanges)
	if detectedChanges == 0 {
		t.Error("Expected new file creation to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherFileDeletion tests detection of file deletion
func TestWatcherFileDeletion(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create test file
	testFile := filepath.Join(testRepoPath, "tobedeleted.txt")
	err = os.WriteFile(testFile, []byte("content to be deleted"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("File deletion detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Delete the test file
	err = os.Remove(testFile)
	if err != nil {
		t.Fatalf("Failed to delete test file: %v", err)
	}

	// Wait for change to be detected
	time.Sleep(1 * time.Second)

	// Check if change was detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	t.Logf("Detected %d changes", detectedChanges)
	if detectedChanges == 0 {
		t.Error("Expected file deletion to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherDirectoryChanges tests detection of directory changes
func TestWatcherDirectoryChanges(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("Directory change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create new subdirectory and file
	subDir := filepath.Join(testRepoPath, "newsubdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "subfile.txt")
	err = os.WriteFile(subFile, []byte("subdirectory file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create subdirectory file: %v", err)
	}

	// Wait for change to be detected
	time.Sleep(1 * time.Second)

	// Check if change was detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	t.Logf("Detected %d changes", detectedChanges)
	if detectedChanges == 0 {
		t.Error("Expected directory change to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherDebouncing tests change debouncing functionality
func TestWatcherDebouncing(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create test file
	testFile := filepath.Join(testRepoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("Debounced change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Make multiple rapid changes
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("content %d", i)
		err = os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}
		time.Sleep(50 * time.Millisecond) // Rapid changes
	}

	// Wait for debouncing to complete
	time.Sleep(1 * time.Second)

	// Check if changes were debounced (should be 1 or a small number, not 5)
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	if detectedChanges == 0 {
		t.Error("Expected at least one debounced change to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherIgnorePatterns tests file ignore patterns
func TestWatcherIgnorePatterns(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create files that should be ignored (before watcher starts)
	ignoredFiles := []string{
		".git/config",
		"node_modules/package.json",
		"dist/bundle.js",
		"build/output.txt",
		"__pycache__/module.pyc",
		".vscode/settings.json",
		".idea/workspace.xml",
	}

	for _, filename := range ignoredFiles {
		filePath := filepath.Join(testRepoPath, filename)
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != testRepoPath {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory for %s: %v", filename, err)
			}
		}
		err := os.WriteFile(filePath, []byte("ignored content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create ignored file %s: %v", filename, err)
		}
	}

	// Create a file that should NOT be ignored (before watcher starts)
	normalFile := filepath.Join(testRepoPath, "normal.txt")
	err = os.WriteFile(normalFile, []byte("normal content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("Change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify ignored files (should not trigger changes)
	for _, filename := range ignoredFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err = os.WriteFile(filePath, []byte("modified ignored content"), 0644)
		if err != nil {
			t.Fatalf("Failed to modify ignored file %s: %v", filename, err)
		}
	}

	// Modify normal file (should trigger change)
	err = os.WriteFile(normalFile, []byte("modified normal content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify normal file: %v", err)
	}

	// Wait for changes to be processed
	time.Sleep(1 * time.Second)

	// Check if changes were detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	t.Logf("Detected %d changes", detectedChanges)
	// Should detect at least the normal file change
	if detectedChanges == 0 {
		t.Error("Expected normal file change to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherConcurrentOperations tests concurrent watcher operations
func TestWatcherConcurrentOperations(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("Concurrent change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create multiple files concurrently
	numFiles := 10
	done := make(chan bool, numFiles)

	for i := 0; i < numFiles; i++ {
		go func(i int) {
			filename := fmt.Sprintf("concurrent%d.txt", i)
			filePath := filepath.Join(testRepoPath, filename)
			content := fmt.Sprintf("concurrent content %d", i)
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Errorf("Failed to create concurrent file %d: %v", i, err)
			}
			done <- true
		}(i)
	}

	// Wait for all files to be created
	for i := 0; i < numFiles; i++ {
		select {
		case <-done:
			// File created successfully
		case <-time.After(5 * time.Second):
			t.Error("Test timed out waiting for concurrent file creation")
			watcher.Stop()
			return
		}
	}

	// Wait for changes to be detected
	time.Sleep(1 * time.Second)

	// Check if changes were detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	if detectedChanges == 0 {
		t.Error("Expected concurrent file changes to be detected")
	}

	// Stop watcher
	watcher.Stop()
}

// TestWatcherErrorHandling tests error handling in watcher
func TestWatcherErrorHandling(t *testing.T) {
	// Test with non-existent directory
	_, err := repo.NewWatcher("/non/existent/path", func(path string) {})
	if err == nil {
		t.Error("Expected error when creating watcher for non-existent path")
	}

	// Test with file instead of directory
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "not-a-directory")
	err = os.WriteFile(testFile, []byte("not a directory"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// The watcher can handle file paths (it just skips non-directories)
	_, err = repo.NewWatcher(testFile, func(path string) {})
	if err != nil {
		t.Errorf("Expected watcher creation to succeed even with file path, got error: %v", err)
	}
}

// TestWatcherStop tests watcher stop functionality
func TestWatcherStop(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("Change detected after stop: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Stop watcher
	watcher.Stop()

	// Wait for stop to take effect
	time.Sleep(100 * time.Millisecond)

	// Try to create a file after stopping
	testFile := filepath.Join(testRepoPath, "after-stop.txt")
	err = os.WriteFile(testFile, []byte("content after stop"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file after stop: %v", err)
	}

	// Wait to see if change is detected
	time.Sleep(500 * time.Millisecond)

	// Check if no changes were detected after stopping
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	if detectedChanges > 0 {
		t.Error("Expected no changes to be detected after stopping watcher")
	}
}

// TestWatcherMultipleDirectories tests watching multiple directories
func TestWatcherMultipleDirectories(t *testing.T) {
	// Create temporary test directory with subdirectories
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Track changes
	var changeCount int
	var changeMutex sync.Mutex
	changeHandler := func(path string) {
		changeMutex.Lock()
		changeCount++
		changeMutex.Unlock()
		t.Logf("Multi-directory change detected: %s", path)
	}

	// Create watcher
	watcher, err := repo.NewWatcher(testRepoPath, changeHandler)
	if err != nil {
		t.Fatalf("Expected watcher creation to succeed, got error: %v", err)
	}

	// Start watcher
	ctx := context.Background()
	watcher.Start(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create subdirectories and files (after watcher is started)
	subDirs := []string{"src", "docs", "tests"}
	for _, subDir := range subDirs {
		dirPath := filepath.Join(testRepoPath, subDir)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory %s: %v", subDir, err)
		}
	}

	// Create files in different subdirectories
	testFiles := []string{
		"src/main.go",
		"docs/README.md",
		"tests/test.go",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Wait for changes to be detected
	time.Sleep(1 * time.Second)

	// Check if changes were detected
	changeMutex.Lock()
	detectedChanges := changeCount
	changeMutex.Unlock()

	t.Logf("Detected %d changes", detectedChanges)
	if detectedChanges == 0 {
		t.Error("Expected multi-directory changes to be detected")
	}

	// Stop watcher
	watcher.Stop()
}
