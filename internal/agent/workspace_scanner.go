package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

const (
	maxScanChars = 100 * 1024
	maxScanFiles = 15
	maxScanDepth = 6
)

// ScanWorkspaceFiles proactively loads source files from a workspace so
// specialist agents can answer questions about a project without the user
// needing to paste code. It first checks for an existing repo index (fast
// path) and falls back to a domain-aware directory scan.
//
// excludePaths lets the caller skip files already present in the prompt
// (e.g. from workspace context or referenced-file detection).
func ScanWorkspaceFiles(workspacePath string, agentType protocol.AgentType, query string, maxChars int, excludePaths map[string]bool) (string, int, error) {
	if workspacePath == "" {
		return "", 0, nil
	}

	info, err := os.Stat(workspacePath)
	if err != nil || !info.IsDir() {
		return "", 0, fmt.Errorf("workspace path is not a valid directory: %s", workspacePath)
	}

	if result, count := tryFromRepoIndex(workspacePath, agentType, query, maxChars, excludePaths); result != "" {
		return result, count, nil
	}

	return scanDirectory(workspacePath, agentType, query, maxChars, excludePaths)
}

// ---------------------------------------------------------------------------
// Tier 1: Repo index
// ---------------------------------------------------------------------------

func tryFromRepoIndex(workspacePath string, agentType protocol.AgentType, query string, maxChars int, excludePaths map[string]bool) (string, int) {
	storage, err := repo.NewStorage()
	if err != nil {
		return "", 0
	}

	cacheKey, err := storage.GetCacheKeyForPath(workspacePath)
	if err != nil {
		return "", 0
	}

	if !storage.IndexExists(cacheKey) {
		return "", 0
	}

	index, err := storage.LoadIndex(cacheKey)
	if err != nil {
		log.Printf("[workspace-scanner] Failed to load repo index for %s: %v", workspacePath, err)
		return "", 0
	}

	files := repo.SearchRelevantFiles(query, index, maxScanFiles*2)
	if len(files) == 0 {
		return "", 0
	}

	extensions := getAgentFileExtensions(agentType)
	extSet := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		extSet[ext] = true
	}

	var sb strings.Builder
	sb.WriteString("\n=== WORKSPACE SOURCE FILES ===\n")
	sb.WriteString("The following source files were loaded from the workspace to give you project context.\n")
	sb.WriteString("Analyze the ACTUAL code below. Each line is prefixed with its real line number.\n\n")

	totalChars := 0
	included := 0

	for _, f := range files {
		if included >= maxScanFiles || totalChars >= maxChars {
			break
		}

		if excludePaths[f.Path] {
			continue
		}

		ext := strings.ToLower(filepath.Ext(f.Path))
		if len(extSet) > 0 && !extSet[ext] && !isEntryPoint(f.Path, agentType) {
			continue
		}

		content, err := repo.DecompressContent(f.Content)
		if err != nil {
			continue
		}

		if totalChars+len(content) > maxChars {
			remaining := maxChars - totalChars
			if remaining < 500 {
				break
			}
			lines := strings.Split(content, "\n")
			var trimmed strings.Builder
			for _, line := range lines {
				if trimmed.Len()+len(line)+1 > remaining {
					break
				}
				if trimmed.Len() > 0 {
					trimmed.WriteByte('\n')
				}
				trimmed.WriteString(line)
			}
			content = trimmed.String() + "\n\n... [TRUNCATED] ..."
		}

		lang := inferLanguage(f.Path)
		numbered := addLineNumbers(content)
		sb.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n%s\n```\n\n", f.Path, lang, lang, numbered))
		totalChars += len(content)
		included++
	}

	if included == 0 {
		return "", 0
	}

	sb.WriteString("=== END WORKSPACE SOURCE FILES ===\n\n")
	return sb.String(), included
}

// ---------------------------------------------------------------------------
// Tier 2: Directory scan
// ---------------------------------------------------------------------------

type scannedFile struct {
	relPath  string
	absPath  string
	size     int64
	priority int // lower = higher priority
}

func scanDirectory(workspacePath string, agentType protocol.AgentType, query string, maxChars int, excludePaths map[string]bool) (string, int, error) {
	extensions := getAgentFileExtensions(agentType)
	extSet := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		extSet[ext] = true
	}

	entryPoints := getEntryPointPatterns(agentType)
	entrySet := make(map[string]bool, len(entryPoints))
	for _, ep := range entryPoints {
		entrySet[strings.ToLower(ep)] = true
	}

	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	var candidates []scannedFile

	absWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", 0, err
	}

	err = filepath.Walk(absWorkspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if repo.ShouldIgnore(info.Name()) {
				return filepath.SkipDir
			}
			rel, relErr := filepath.Rel(absWorkspace, path)
			if relErr != nil {
				return nil
			}
			if rel != "." && strings.Count(rel, string(filepath.Separator)) >= maxScanDepth {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		baseName := strings.ToLower(info.Name())

		if binaryExtensions[ext] {
			return nil
		}

		// For typed agents, only include matching extensions + entry points
		matchesExt := len(extSet) == 0 || extSet[ext]
		matchesEntry := entrySet[baseName]
		if !matchesExt && !matchesEntry {
			return nil
		}

		if info.Size() > maxFileSize {
			return nil
		}

		rel, err := filepath.Rel(absWorkspace, path)
		if err != nil {
			return nil
		}

		if excludePaths[rel] || excludePaths[path] {
			return nil
		}

		priority := 100
		if matchesEntry {
			priority = 10
		}
		relLower := strings.ToLower(rel)
		for _, w := range queryWords {
			if len(w) >= 3 && strings.Contains(relLower, w) {
				priority -= 20
				break
			}
		}
		if strings.Contains(relLower, "test") || strings.Contains(relLower, "spec") {
			priority += 30
		}

		candidates = append(candidates, scannedFile{
			relPath:  rel,
			absPath:  path,
			size:     info.Size(),
			priority: priority,
		})

		return nil
	})
	if err != nil {
		return "", 0, fmt.Errorf("directory walk failed: %w", err)
	}

	if len(candidates) == 0 {
		return "", 0, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].size < candidates[j].size
	})

	var sb strings.Builder
	sb.WriteString("\n=== WORKSPACE SOURCE FILES ===\n")
	sb.WriteString("The following source files were loaded from the workspace to give you project context.\n")
	sb.WriteString("Analyze the ACTUAL code below. Each line is prefixed with its real line number.\n\n")

	totalChars := 0
	included := 0

	for _, c := range candidates {
		if included >= maxScanFiles || totalChars >= maxChars {
			break
		}

		content, _, err := ReadFileForPrompt(c.relPath, workspacePath)
		if err != nil {
			continue
		}

		raw := stripLineNumbers(content)

		if totalChars+len(raw) > maxChars {
			remaining := maxChars - totalChars
			if remaining < 500 {
				break
			}
			lines := strings.Split(raw, "\n")
			var trimmed strings.Builder
			for _, line := range lines {
				if trimmed.Len()+len(line)+1 > remaining {
					break
				}
				if trimmed.Len() > 0 {
					trimmed.WriteByte('\n')
				}
				trimmed.WriteString(line)
			}
			content = addLineNumbers(trimmed.String()) + "\n\n... [TRUNCATED] ..."
		}

		lang := inferLanguage(c.relPath)
		sb.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n%s\n```\n\n", c.relPath, lang, lang, content))
		totalChars += len(raw)
		included++
	}

	if included == 0 {
		return "", 0, nil
	}

	remaining := len(candidates) - included
	if remaining > 0 {
		sb.WriteString(fmt.Sprintf("(%d additional matching files not shown)\n\n", remaining))
	}

	sb.WriteString("=== END WORKSPACE SOURCE FILES ===\n\n")
	return sb.String(), included, nil
}

// stripLineNumbers removes line-number prefixes added by ReadFileForPrompt
// so we can measure raw content size accurately.
func stripLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	var sb strings.Builder
	sb.Grow(len(content))
	for i, line := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if idx := strings.Index(line, " | "); idx != -1 && idx < 8 {
			sb.WriteString(line[idx+3:])
		} else {
			sb.WriteString(line)
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Domain mappings
// ---------------------------------------------------------------------------

func getAgentFileExtensions(agentType protocol.AgentType) []string {
	switch agentType {
	case protocol.AgentTypeRust:
		return []string{".rs", ".toml", ".lock"}
	case protocol.AgentTypeBackend:
		return []string{".go", ".mod", ".sum"}
	case protocol.AgentTypeFrontend:
		return []string{".ts", ".tsx", ".js", ".jsx", ".css", ".scss", ".html", ".vue", ".svelte"}
	case protocol.AgentTypeDevOps:
		return []string{".yaml", ".yml", ".tf", ".hcl", ".dockerfile", ".sh", ".bash"}
	case protocol.AgentTypeDatabase:
		return []string{".sql", ".go", ".py", ".ts"}
	case protocol.AgentTypeSecurity:
		return nil // all extensions
	default:
		return nil // all extensions
	}
}

func getEntryPointPatterns(agentType protocol.AgentType) []string {
	common := []string{"readme.md", "makefile"}
	switch agentType {
	case protocol.AgentTypeRust:
		return append(common, "main.rs", "lib.rs", "cargo.toml", "mod.rs", "build.rs")
	case protocol.AgentTypeBackend:
		return append(common, "main.go", "go.mod", "server.go", "app.go", "handler.go")
	case protocol.AgentTypeFrontend:
		return append(common, "app.tsx", "app.jsx", "index.ts", "index.tsx", "index.js", "package.json", "vite.config.ts", "tsconfig.json")
	case protocol.AgentTypeDevOps:
		return append(common, "dockerfile", "docker-compose.yml", "docker-compose.yaml", ".github/workflows", "terraform.tf", "main.tf")
	case protocol.AgentTypeDatabase:
		return append(common, "schema.sql", "migrations", "models.go", "models.py")
	default:
		return common
	}
}

func isEntryPoint(filePath string, agentType protocol.AgentType) bool {
	baseName := strings.ToLower(filepath.Base(filePath))
	for _, ep := range getEntryPointPatterns(agentType) {
		if baseName == ep {
			return true
		}
	}
	return false
}
