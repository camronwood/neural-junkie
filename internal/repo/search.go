package repo

import (
	"sort"
	"strings"
)

// SearchResult represents a file matching a search query with a relevance score
type SearchResult struct {
	File  *SourceFile
	Score int
}

// SearchRelevantFiles searches for source files relevant to a query
// Returns up to maxFiles files, sorted by relevance
func SearchRelevantFiles(query string, index *RepositoryIndex, maxFiles int) []*SourceFile {
	if index == nil || index.SourceFiles == nil || len(index.SourceFiles) == 0 {
		return []*SourceFile{}
	}

	// Normalize query
	query = strings.ToLower(query)
	keywords := extractKeywords(query)

	// Score each file
	results := []SearchResult{}
	for _, file := range index.SourceFiles {
		score := scoreFile(file, query, keywords)
		if score > 0 {
			results = append(results, SearchResult{
				File:  file,
				Score: score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top N files
	limit := maxFiles
	if limit > len(results) {
		limit = len(results)
	}

	files := make([]*SourceFile, limit)
	for i := 0; i < limit; i++ {
		files[i] = results[i].File
	}

	return files
}

// scoreFile calculates a relevance score for a file based on the query
func scoreFile(file *SourceFile, query string, keywords []string) int {
	score := 0

	lowerPath := strings.ToLower(file.Path)
	fileName := strings.ToLower(getFileName(file.Path))

	// Exact filename match (highest priority)
	for _, keyword := range keywords {
		if fileName == keyword || fileName == keyword+"."+file.Language {
			score += 100
		}
	}

	// Filename contains keyword
	for _, keyword := range keywords {
		if strings.Contains(fileName, keyword) {
			score += 50
		}
	}

	// Path contains keyword
	for _, keyword := range keywords {
		if strings.Contains(lowerPath, keyword) {
			score += 20
		}
	}

	// Language match (if query mentions a language)
	if strings.Contains(query, strings.ToLower(file.Language)) {
		score += 10
	}

	// Bonus for common entry points
	if isEntryPoint(fileName) {
		score += 30
	}

	// Bonus for configuration files
	if isConfigFile(fileName) {
		score += 15
	}

	return score
}

// extractKeywords extracts meaningful keywords from a query
func extractKeywords(query string) []string {
	// Remove common question words
	stopWords := []string{
		"how", "what", "where", "when", "why", "who",
		"does", "is", "are", "can", "the", "a", "an",
		"in", "on", "at", "to", "for", "of", "with",
		"this", "that", "these", "those",
	}

	words := strings.Fields(query)
	keywords := []string{}

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:"))

		// Skip stop words and short words
		if len(word) < 3 {
			continue
		}

		isStopWord := false
		for _, stopWord := range stopWords {
			if word == stopWord {
				isStopWord = true
				break
			}
		}

		if !isStopWord {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// getFileName extracts the filename from a path
func getFileName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

// isEntryPoint checks if a file is likely an entry point
func isEntryPoint(fileName string) bool {
	entryPoints := []string{
		"main", "index", "app", "server", "client",
		"init", "bootstrap", "startup", "entry",
	}

	fileNameLower := strings.ToLower(fileName)
	for _, entry := range entryPoints {
		if strings.HasPrefix(fileNameLower, entry) {
			return true
		}
	}

	return false
}

// isConfigFile checks if a file is a configuration file
func isConfigFile(fileName string) bool {
	configPatterns := []string{
		"config", "settings", "env", "constants",
		"options", "preferences", "setup",
	}

	fileNameLower := strings.ToLower(fileName)
	for _, pattern := range configPatterns {
		if strings.Contains(fileNameLower, pattern) {
			return true
		}
	}

	return false
}

// SearchByPath searches for files matching a specific path pattern
func SearchByPath(pattern string, index *RepositoryIndex) []*SourceFile {
	if index == nil || index.SourceFiles == nil {
		return []*SourceFile{}
	}

	pattern = strings.ToLower(pattern)
	results := []*SourceFile{}

	for _, file := range index.SourceFiles {
		if strings.Contains(strings.ToLower(file.Path), pattern) {
			results = append(results, file)
		}
	}

	return results
}

// SearchByLanguage returns all files of a specific language
func SearchByLanguage(language string, index *RepositoryIndex) []*SourceFile {
	if index == nil || index.SourceFiles == nil {
		return []*SourceFile{}
	}

	language = strings.ToLower(language)
	results := []*SourceFile{}

	for _, file := range index.SourceFiles {
		if strings.ToLower(file.Language) == language {
			results = append(results, file)
		}
	}

	return results
}
