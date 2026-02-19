package protocol

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectedPath represents a file path found in a message
type DetectedPath struct {
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	Exists      bool   `json:"exists"`
	Context     string `json:"context"` // surrounding text
}

// PathDetectionResult contains all detected paths in a message
type PathDetectionResult struct {
	Paths []DetectedPath `json:"paths"`
	Found bool           `json:"found"`
}

// DetectLocalPaths finds and validates local file paths in a message
func DetectLocalPaths(content string) *PathDetectionResult {
	result := &PathDetectionResult{
		Paths: []DetectedPath{},
		Found:  false,
	}

	// Regex patterns for different path formats
	patterns := []string{
		`/[^\s]+`,                    // Unix absolute paths
		`[A-Za-z]:\\[^\s]*`,          // Windows absolute paths
		`~/[^\s]+`,                   // Unix home directory paths
	}

	// Combine patterns
	combinedPattern := strings.Join(patterns, "|")
	re := regexp.MustCompile(combinedPattern)

	matches := re.FindAllString(content, -1)
	
	for _, match := range matches {
		// Clean up the match
		path := strings.TrimSpace(match)
		
		// Skip if too short or looks like a command
		if len(path) < 3 || strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
			continue
		}

		// Expand home directory if needed
		if strings.HasPrefix(path, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				path = filepath.Join(homeDir, path[2:])
			}
		}

		// Check if path exists
		info, err := os.Stat(path)
		exists := err == nil
		isDir := exists && info.IsDir()

		// Only include if it exists and is a directory (repository)
		if exists && isDir {
			// Find context around the path
			context := extractContext(content, match)
			
			detectedPath := DetectedPath{
				Path:        path,
				IsDirectory: isDir,
				Exists:      exists,
				Context:     context,
			}
			
			result.Paths = append(result.Paths, detectedPath)
			result.Found = true
		}
	}

	return result
}

// extractContext gets the surrounding text around a path match
func extractContext(content, match string) string {
	index := strings.Index(content, match)
	if index == -1 {
		return ""
	}

	// Get 50 characters before and after
	start := index - 50
	if start < 0 {
		start = 0
	}
	
	end := index + len(match) + 50
	if end > len(content) {
		end = len(content)
	}

	context := content[start:end]
	return strings.TrimSpace(context)
}

// IsRepositoryPath checks if a path looks like a repository
func IsRepositoryPath(path string) bool {
	// Check for common repository indicators
	repoIndicators := []string{
		".git",
		"package.json",
		"go.mod",
		"requirements.txt",
		"pom.xml",
		"build.gradle",
		"Cargo.toml",
	}

	for _, indicator := range repoIndicators {
		if _, err := os.Stat(filepath.Join(path, indicator)); err == nil {
			return true
		}
	}

	// Check if it's a directory with source files
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	// Look for source code files
	sourceExtensions := []string{".go", ".js", ".ts", ".py", ".java", ".rs", ".cpp", ".c", ".h"}
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			for _, sourceExt := range sourceExtensions {
				if ext == sourceExt {
					return true
				}
			}
		}
	}

	return false
}

// GetBestPathForRepository returns the best path to use for repository analysis
func GetBestPathForRepository(detectedPaths []DetectedPath) *DetectedPath {
	if len(detectedPaths) == 0 {
		return nil
	}

	// If only one path, return it
	if len(detectedPaths) == 1 {
		return &detectedPaths[0]
	}

	// Prefer paths that look more like repositories
	bestPath := &detectedPaths[0]
	bestScore := 0

	for i := range detectedPaths {
		score := 0
		path := detectedPaths[i].Path

		// Score based on repository indicators
		if IsRepositoryPath(path) {
			score += 10
		}

		// Prefer longer paths (more specific)
		score += len(strings.Split(path, string(filepath.Separator)))

		// Prefer paths with "src", "app", "lib" in the name
		pathLower := strings.ToLower(path)
		if strings.Contains(pathLower, "src") || 
		   strings.Contains(pathLower, "app") || 
		   strings.Contains(pathLower, "lib") {
			score += 5
		}

		if score > bestScore {
			bestScore = score
			bestPath = &detectedPaths[i]
		}
	}

	return bestPath
}
