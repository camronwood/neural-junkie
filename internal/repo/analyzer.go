package repo

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxFileSize  = 1024 * 1024       // 1MB - max file size to read
	MaxTotalSize = 100 * 1024 * 1024 // 100MB - max total repo size
	MaxFiles     = 10000             // Max number of files to index
)

// Analyzer analyzes and indexes repositories
type Analyzer struct {
	progressCallback func(progress int, message string)
}

// NewAnalyzer creates a new repository analyzer
func NewAnalyzer(progressCallback func(progress int, message string)) *Analyzer {
	return &Analyzer{
		progressCallback: progressCallback,
	}
}

// AnalyzeRepository analyzes a repository and creates an index
func (a *Analyzer) AnalyzeRepository(ctx context.Context, repoPath string) (*RepositoryIndex, error) {
	// Validate path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("repository path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repository path is not a directory")
	}

	a.updateProgress(0, "Starting repository analysis...")

	index := &RepositoryIndex{
		Path:         absPath,
		Name:         filepath.Base(absPath),
		LastIndexed:  time.Now(),
		KeyFiles:     make(map[string]string),
		Dependencies: make(map[string][]string),
		CodePatterns: []string{},
		FileModTimes: make(map[string]time.Time),
		SourceFiles:  make(map[string]*SourceFile),
	}

	// Step 1: Build directory structure
	a.updateProgress(10, "Scanning directory structure...")
	structure, fileCount, totalSize, err := a.buildDirectoryStructure(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build directory structure: %w", err)
	}
	index.Structure = structure
	index.FileCount = fileCount
	index.TotalSize = totalSize

	// Check size limits
	if totalSize > MaxTotalSize {
		return nil, fmt.Errorf("repository too large: %d bytes (max %d)", totalSize, MaxTotalSize)
	}
	if fileCount > MaxFiles {
		return nil, fmt.Errorf("too many files: %d (max %d)", fileCount, MaxFiles)
	}

	// Step 2: Extract key files
	a.updateProgress(20, "Extracting key files...")
	if err := a.extractKeyFiles(ctx, absPath, index); err != nil {
		return nil, fmt.Errorf("failed to extract key files: %w", err)
	}

	// Step 3: Index source files
	a.updateProgress(30, "Indexing source code files...")
	if err := a.indexSourceFiles(ctx, absPath, index); err != nil {
		return nil, fmt.Errorf("failed to index source files: %w", err)
	}

	// Step 4: Parse dependencies
	a.updateProgress(60, "Parsing dependencies...")
	if err := a.parseDependencies(ctx, absPath, index); err != nil {
		// Non-fatal error, log and continue
		fmt.Printf("Warning: failed to parse dependencies: %v\n", err)
	}

	// Step 5: Get git information
	a.updateProgress(75, "Reading git history...")
	gitInfo, err := a.getGitInfo(ctx, absPath)
	if err != nil {
		// Non-fatal error, log and continue
		fmt.Printf("Warning: failed to get git info: %v\n", err)
	} else {
		index.GitInfo = gitInfo
	}

	// Step 6: Identify code patterns
	a.updateProgress(85, "Identifying code patterns...")
	index.CodePatterns = a.identifyCodePatterns(index)

	// Step 7: Generate architecture overview
	a.updateProgress(95, "Generating architecture overview...")
	index.ArchitectureDoc = a.generateArchitectureDoc(index)

	a.updateProgress(100, "Analysis complete!")

	TrimRepositoryIndexFootprint(index)

	return index, nil
}

// buildDirectoryStructure recursively builds the directory tree
func (a *Analyzer) buildDirectoryStructure(ctx context.Context, path string) (*DirectoryNode, int, int64, error) {
	fileCount := 0
	var totalSize int64

	var build func(string, int) (*DirectoryNode, error)
	build = func(currentPath string, depth int) (*DirectoryNode, error) {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Limit recursion depth
		if depth > 20 {
			return nil, nil
		}

		info, err := os.Stat(currentPath)
		if err != nil {
			return nil, err
		}

		relPath, _ := filepath.Rel(path, currentPath)
		node := &DirectoryNode{
			Name:        filepath.Base(currentPath),
			Path:        relPath,
			IsDirectory: info.IsDir(),
		}

		if !info.IsDir() {
			node.Size = info.Size()
			totalSize += info.Size()
			fileCount++

			// Determine language
			ext := filepath.Ext(currentPath)
			if lang, ok := LanguageExtensions[ext]; ok {
				node.Language = lang
			}

			return node, nil
		}

		// Read directory contents
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			// Skip hidden files and common ignore patterns
			name := entry.Name()
			if strings.HasPrefix(name, ".") && name != ".env.example" {
				continue
			}
			if ShouldIgnore(name) {
				continue
			}

			childPath := filepath.Join(currentPath, name)
			child, err := build(childPath, depth+1)
			if err != nil {
				continue // Skip problematic files
			}
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node, nil
	}

	root, err := build(path, 0)
	return root, fileCount, totalSize, err
}

// extractKeyFiles extracts content from important files
func (a *Analyzer) extractKeyFiles(ctx context.Context, repoPath string, index *RepositoryIndex) error {
	for _, keyFile := range KeyFileTypes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		filePath := filepath.Join(repoPath, keyFile)
		info, err := os.Stat(filePath)
		if err == nil {
			content, err := readFileContent(filePath, MaxFileSize)
			if err != nil {
				continue
			}
			index.KeyFiles[keyFile] = content
			index.FileModTimes[keyFile] = info.ModTime()
		}
	}
	return nil
}

// indexSourceFiles indexes all source code files with compression
func (a *Analyzer) indexSourceFiles(ctx context.Context, repoPath string, index *RepositoryIndex) error {
	stats := &CompressionStats{}
	sourceFileCount := 0
	cappedLogged := false

	// Walk the directory tree and collect source files
	var walkSourceFiles func(*DirectoryNode, string) error
	walkSourceFiles = func(node *DirectoryNode, currentPath string) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !node.IsDirectory {
			// Check if it's a source file (has a language)
			if node.Language != "" {
				filePath := filepath.Join(repoPath, node.Path)

				// Read file info
				info, err := os.Stat(filePath)
				if err != nil {
					return nil // Skip files we can't read
				}

				// Skip files that are too large
				if info.Size() > MaxFileSize {
					return nil
				}

				// Read file content
				content, err := readFileContent(filePath, MaxFileSize)
				if err != nil {
					return nil // Skip files we can't read
				}

				// Compress content
				compressed, compressedSize, err := CompressContent(content)
				if err != nil {
					fmt.Printf("Warning: failed to compress %s: %v\n", node.Path, err)
					return nil
				}

				// Create source file entry
				sourceFile := &SourceFile{
					Path:           node.Path,
					Language:       node.Language,
					Size:           info.Size(),
					CompressedSize: compressedSize,
					Content:        compressed,
					ModTime:        info.ModTime(),
				}

				// Add to index
				if len(index.SourceFiles) >= MaxIndexedSourceFilesInMemory {
					if !cappedLogged {
						log.Printf("[repo] Source file index capped at %d files (memory); skipping further file bodies", MaxIndexedSourceFilesInMemory)
						cappedLogged = true
					}
					return nil
				}
				index.SourceFiles[node.Path] = sourceFile

				// Update stats
				stats.OriginalSize += info.Size()
				stats.CompressedSize += compressedSize
				stats.FileCount++
				sourceFileCount++

				// Update progress every 10 files
				if sourceFileCount%10 == 0 {
					progress := 30 + int(float64(sourceFileCount)/float64(index.FileCount)*30)
					if progress > 60 {
						progress = 60
					}
					a.updateProgress(progress, fmt.Sprintf("Indexed %d source files...", sourceFileCount))
				}
			}
			return nil
		}

		// Recursively process children
		for _, child := range node.Children {
			if err := walkSourceFiles(child, filepath.Join(currentPath, node.Name)); err != nil {
				return err
			}
		}

		return nil
	}

	// Start walking from the root
	if index.Structure != nil {
		if err := walkSourceFiles(index.Structure, repoPath); err != nil {
			return err
		}
	}

	// Log compression statistics
	if stats.FileCount > 0 {
		fmt.Printf("Source file indexing complete:\n")
		fmt.Printf("  Files indexed: %d\n", stats.FileCount)
		fmt.Printf("  Original size: %s\n", FormatSize(stats.OriginalSize))
		fmt.Printf("  Compressed size: %s\n", FormatSize(stats.CompressedSize))
		fmt.Printf("  Space saved: %s (%.1f%% compression)\n",
			FormatSize(stats.SpaceSaved()), 100-stats.CompressionRatio())
	}

	return nil
}

// parseDependencies parses dependency files
func (a *Analyzer) parseDependencies(ctx context.Context, repoPath string, index *RepositoryIndex) error {
	// Parse package.json (Node.js)
	if content, ok := index.KeyFiles["package.json"]; ok {
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal([]byte(content), &pkg); err == nil {
			deps := []string{}
			for name := range pkg.Dependencies {
				deps = append(deps, name)
			}
			index.Dependencies["npm"] = deps
		}
	}

	// Parse go.mod (Go)
	if content, ok := index.KeyFiles["go.mod"]; ok {
		deps := []string{}
		scanner := bufio.NewScanner(strings.NewReader(content))
		inRequire := false
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "require") {
				inRequire = true
				continue
			}
			if inRequire {
				if line == ")" {
					break
				}
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					deps = append(deps, parts[0])
				}
			}
		}
		index.Dependencies["go"] = deps
	}

	// Parse requirements.txt (Python)
	reqPath := filepath.Join(repoPath, "requirements.txt")
	if content, err := readFileContent(reqPath, MaxFileSize); err == nil {
		deps := []string{}
		scanner := bufio.NewScanner(strings.NewReader(content))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				// Extract package name before version specifier
				parts := strings.FieldsFunc(line, func(r rune) bool {
					return r == '=' || r == '>' || r == '<' || r == '~'
				})
				if len(parts) > 0 {
					deps = append(deps, parts[0])
				}
			}
		}
		index.Dependencies["pip"] = deps
	}

	return nil
}

// getGitInfo retrieves git repository information
func (a *Analyzer) getGitInfo(ctx context.Context, repoPath string) (*GitInfo, error) {
	// Check if it's a git repository
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	gitInfo := &GitInfo{}

	// Get current branch
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if output, err := cmd.Output(); err == nil {
		gitInfo.Branch = strings.TrimSpace(string(output))
	}

	// Get last commit info
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--pretty=format:%H||%an||%at||%s")
	if output, err := cmd.Output(); err == nil {
		parts := strings.Split(string(output), "||")
		if len(parts) == 4 {
			gitInfo.LastCommit = parts[0]
			gitInfo.LastCommitMsg = parts[3]
			// Parse timestamp
			var timestamp int64
			fmt.Sscanf(parts[2], "%d", &timestamp)
			gitInfo.LastCommitDate = time.Unix(timestamp, 0)
		}
	}

	// Get recent commits
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-10", "--pretty=format:%H||%an||%at||%s")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			parts := strings.Split(line, "||")
			if len(parts) == 4 {
				var timestamp int64
				fmt.Sscanf(parts[2], "%d", &timestamp)
				commit := CommitInfo{
					Hash:    parts[0][:8], // Short hash
					Author:  parts[1],
					Date:    time.Unix(timestamp, 0),
					Message: parts[3],
				}
				gitInfo.RecentCommits = append(gitInfo.RecentCommits, commit)
			}
		}
	}

	return gitInfo, nil
}

// identifyCodePatterns identifies frameworks and patterns in the codebase
func (a *Analyzer) identifyCodePatterns(index *RepositoryIndex) []string {
	patterns := []string{}

	// Check package.json for frameworks
	if content, ok := index.KeyFiles["package.json"]; ok {
		if strings.Contains(content, "\"react\"") {
			patterns = append(patterns, "React")
		}
		if strings.Contains(content, "\"vue\"") {
			patterns = append(patterns, "Vue.js")
		}
		if strings.Contains(content, "\"angular\"") {
			patterns = append(patterns, "Angular")
		}
		if strings.Contains(content, "\"next\"") {
			patterns = append(patterns, "Next.js")
		}
		if strings.Contains(content, "\"express\"") {
			patterns = append(patterns, "Express.js")
		}
	}

	// Check go.mod for frameworks
	if content, ok := index.KeyFiles["go.mod"]; ok {
		if strings.Contains(content, "github.com/gin-gonic/gin") {
			patterns = append(patterns, "Gin (Go)")
		}
		if strings.Contains(content, "github.com/gofiber/fiber") {
			patterns = append(patterns, "Fiber (Go)")
		}
	}

	// Check for containerization
	if _, ok := index.KeyFiles["Dockerfile"]; ok {
		patterns = append(patterns, "Docker")
	}
	if _, ok := index.KeyFiles["docker-compose.yml"]; ok {
		patterns = append(patterns, "Docker Compose")
	}

	return patterns
}

// generateArchitectureDoc generates a high-level architecture overview
func (a *Analyzer) generateArchitectureDoc(index *RepositoryIndex) string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("# %s - Repository Overview\n\n", index.Name))
	doc.WriteString(fmt.Sprintf("**Last Indexed:** %s\n\n", index.LastIndexed.Format("2006-01-02 15:04:05")))

	// Repository statistics
	doc.WriteString("## Statistics\n\n")
	doc.WriteString(fmt.Sprintf("- **Files:** %d\n", index.FileCount))
	doc.WriteString(fmt.Sprintf("- **Total Size:** %.2f MB\n", float64(index.TotalSize)/(1024*1024)))

	if index.GitInfo != nil {
		doc.WriteString(fmt.Sprintf("- **Branch:** %s\n", index.GitInfo.Branch))
		doc.WriteString(fmt.Sprintf("- **Last Commit:** %s\n", index.GitInfo.LastCommitMsg))
	}
	doc.WriteString("\n")

	// Code patterns/frameworks
	if len(index.CodePatterns) > 0 {
		doc.WriteString("## Technologies & Frameworks\n\n")
		for _, pattern := range index.CodePatterns {
			doc.WriteString(fmt.Sprintf("- %s\n", pattern))
		}
		doc.WriteString("\n")
	}

	// Dependencies
	if len(index.Dependencies) > 0 {
		doc.WriteString("## Dependencies\n\n")
		for mgr, deps := range index.Dependencies {
			doc.WriteString(fmt.Sprintf("**%s** (%d packages)\n", mgr, len(deps)))
		}
		doc.WriteString("\n")
	}

	// Key files
	if len(index.KeyFiles) > 0 {
		doc.WriteString("## Key Files\n\n")
		for filename := range index.KeyFiles {
			doc.WriteString(fmt.Sprintf("- %s\n", filename))
		}
		doc.WriteString("\n")
	}

	return doc.String()
}

// updateProgress calls the progress callback if set
func (a *Analyzer) updateProgress(progress int, message string) {
	if a.progressCallback != nil {
		a.progressCallback(progress, message)
	}
}

// readFileContent reads a file's content with size limit
func readFileContent(filePath string, maxSize int64) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	if info.Size() > maxSize {
		return "", fmt.Errorf("file too large: %d bytes", info.Size())
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// ShouldIgnore checks if a file/directory should be ignored during scanning.
func ShouldIgnore(name string) bool {
	ignorePatterns := []string{
		"node_modules",
		"vendor",
		"target",
		"dist",
		"build",
		"__pycache__",
		".git",
		".svn",
		".hg",
		"venv",
		"env",
		".venv",
		".env",
		"bin",
		"obj",
		".idea",
		".vscode",
		".DS_Store",
	}

	for _, pattern := range ignorePatterns {
		if name == pattern {
			return true
		}
	}

	return false
}

// IsIndexStale checks if a cached index needs to be refreshed
func (a *Analyzer) IsIndexStale(ctx context.Context, repoPath string, index *RepositoryIndex) (bool, string, error) {
	// Check if git commit has changed
	gitInfo, err := a.getGitInfo(ctx, repoPath)
	isGitRepo := err == nil

	if isGitRepo && index.GitInfo != nil {
		if gitInfo.LastCommit != index.GitInfo.LastCommit {
			return true, "Git commit changed", nil
		}
	}

	// Check if file count has changed significantly
	currentFileCount := 0
	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !ShouldIgnore(info.Name()) {
			currentFileCount++
		}
		return nil
	})

	if abs(currentFileCount-index.FileCount) > index.FileCount/10 {
		return true, fmt.Sprintf("File count changed significantly (%d -> %d)", index.FileCount, currentFileCount), nil
	}

	// Check if any key files have been modified
	for filename := range index.KeyFiles {
		filePath := filepath.Join(repoPath, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			// File deleted
			return true, fmt.Sprintf("Key file deleted: %s", filename), nil
		}

		if oldModTime, ok := index.FileModTimes[filename]; ok {
			if info.ModTime().After(oldModTime) {
				return true, fmt.Sprintf("Key file modified: %s", filename), nil
			}
		}
	}

	// Check if any source files have been modified (not just key files)
	for filePath, sourceFile := range index.SourceFiles {
		fullPath := filepath.Join(repoPath, filePath)
		info, err := os.Stat(fullPath)
		if err != nil {
			// File deleted
			return true, fmt.Sprintf("Source file deleted: %s", filePath), nil
		}

		if info.ModTime().After(sourceFile.ModTime) {
			return true, fmt.Sprintf("Source file modified: %s", filePath), nil
		}
	}

	// Check for new files that aren't in the index
	keyFileSet := make(map[string]struct{}, len(KeyFileTypes))
	for _, keyFile := range KeyFileTypes {
		keyFileSet[keyFile] = struct{}{}
	}
	newFilesFound := false
	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if ShouldIgnore(info.Name()) {
			return nil
		}

		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}

		// Check if this file is in our index
		_, inKeyFiles := index.KeyFiles[relPath]
		_, inSourceFiles := index.SourceFiles[relPath]

		if !inKeyFiles && !inSourceFiles {
			// Ignore non-indexed assets so they don't force unnecessary reindexing.
			if _, isKeyFile := keyFileSet[filepath.Base(relPath)]; !isKeyFile {
				if _, isSourceLike := LanguageExtensions[strings.ToLower(filepath.Ext(relPath))]; !isSourceLike {
					return nil
				}
			}
			newFilesFound = true
			return filepath.SkipDir // Stop walking this directory
		}
		return nil
	})

	if newFilesFound {
		return true, "New files detected", nil
	}

	// Use different cache expiry based on whether it's a git repo
	var maxAge time.Duration
	if isGitRepo {
		maxAge = 7 * 24 * time.Hour // 7 days for git repos
	} else {
		maxAge = 24 * time.Hour // 24 hours for non-git repos (like meeting notes)
	}

	if time.Since(index.LastIndexed) > maxAge {
		return true, fmt.Sprintf("Index older than %v", maxAge), nil
	}

	return false, "", nil
}

// IncrementalAnalyze performs incremental analysis based on an existing index
func (a *Analyzer) IncrementalAnalyze(ctx context.Context, repoPath string, oldIndex *RepositoryIndex) (*RepositoryIndex, error) {
	a.updateProgress(0, "Starting incremental analysis...")

	// Start with the old index
	index := &RepositoryIndex{
		Path:         oldIndex.Path,
		Name:         oldIndex.Name,
		LastIndexed:  time.Now(),
		KeyFiles:     make(map[string]string),
		Dependencies: make(map[string][]string),
		CodePatterns: oldIndex.CodePatterns,
		FileModTimes: make(map[string]time.Time),
		SourceFiles:  make(map[string]*SourceFile),
	}

	// Copy old file mod times
	for k, v := range oldIndex.FileModTimes {
		index.FileModTimes[k] = v
	}

	// Copy old source files
	for k, v := range oldIndex.SourceFiles {
		index.SourceFiles[k] = v
	}

	a.updateProgress(10, "Scanning for changes...")

	// Rebuild directory structure (fast operation)
	structure, fileCount, totalSize, err := a.buildDirectoryStructure(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build directory structure: %w", err)
	}
	index.Structure = structure
	index.FileCount = fileCount
	index.TotalSize = totalSize

	a.updateProgress(30, "Updating key files...")

	// Re-extract key files (only if modified)
	for _, keyFile := range KeyFileTypes {
		filePath := filepath.Join(repoPath, keyFile)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// Check if file was modified
		if oldModTime, ok := oldIndex.FileModTimes[keyFile]; ok {
			if !info.ModTime().After(oldModTime) {
				// File unchanged, copy from old index
				if oldContent, ok := oldIndex.KeyFiles[keyFile]; ok {
					index.KeyFiles[keyFile] = oldContent
					continue
				}
			}
		}

		// File is new or modified, read it
		content, err := readFileContent(filePath, MaxFileSize)
		if err == nil {
			index.KeyFiles[keyFile] = content
			index.FileModTimes[keyFile] = info.ModTime()
		}
	}

	a.updateProgress(40, "Updating source files...")

	// Update source files incrementally
	updatedFiles := 0

	// Walk the directory tree to find changed/new files
	var walkSourceFiles func(*DirectoryNode, string) error
	walkSourceFiles = func(node *DirectoryNode, currentPath string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !node.IsDirectory && node.Language != "" {
			filePath := filepath.Join(repoPath, node.Path)
			info, err := os.Stat(filePath)
			if err != nil {
				return nil
			}

			if info.Size() > MaxFileSize {
				return nil
			}

			// Check if file is new or modified
			if oldFile, exists := oldIndex.SourceFiles[node.Path]; exists {
				if !info.ModTime().After(oldFile.ModTime) {
					// File unchanged, already copied
					return nil
				}
			}

			// File is new or modified, reindex it
			content, err := readFileContent(filePath, MaxFileSize)
			if err != nil {
				return nil
			}

			compressed, compressedSize, err := CompressContent(content)
			if err != nil {
				return nil
			}

			index.SourceFiles[node.Path] = &SourceFile{
				Path:           node.Path,
				Language:       node.Language,
				Size:           info.Size(),
				CompressedSize: compressedSize,
				Content:        compressed,
				ModTime:        info.ModTime(),
			}

			updatedFiles++
		}

		for _, child := range node.Children {
			if err := walkSourceFiles(child, filepath.Join(currentPath, node.Name)); err != nil {
				return err
			}
		}

		return nil
	}

	if index.Structure != nil {
		if err := walkSourceFiles(index.Structure, repoPath); err != nil {
			return nil, err
		}
	}

	// Remove deleted files from index
	currentPaths := make(map[string]bool)
	var collectPaths func(*DirectoryNode)
	collectPaths = func(node *DirectoryNode) {
		if !node.IsDirectory {
			currentPaths[node.Path] = true
		}
		for _, child := range node.Children {
			collectPaths(child)
		}
	}
	if index.Structure != nil {
		collectPaths(index.Structure)
	}

	for path := range index.SourceFiles {
		if !currentPaths[path] {
			delete(index.SourceFiles, path)
		}
	}

	fmt.Printf("Incremental update: %d source files updated\n", updatedFiles)

	a.updateProgress(60, "Parsing dependencies...")

	// Re-parse dependencies (uses cached key files)
	if err := a.parseDependencies(ctx, repoPath, index); err != nil {
		fmt.Printf("Warning: failed to parse dependencies: %v\n", err)
	}

	a.updateProgress(70, "Reading git history...")

	// Update git information
	gitInfo, err := a.getGitInfo(ctx, repoPath)
	if err != nil {
		fmt.Printf("Warning: failed to get git info: %v\n", err)
	} else {
		index.GitInfo = gitInfo
	}

	a.updateProgress(85, "Identifying code patterns...")

	// Re-identify code patterns
	index.CodePatterns = a.identifyCodePatterns(index)

	a.updateProgress(95, "Generating architecture overview...")

	// Regenerate architecture doc
	index.ArchitectureDoc = a.generateArchitectureDoc(index)

	a.updateProgress(100, "Incremental analysis complete!")

	TrimRepositoryIndexFootprint(index)

	return index, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
