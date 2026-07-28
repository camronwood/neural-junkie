package agent

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/codeintel"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

var (
	codebaseMentionRE     = regexp.MustCompile(`(?i)@codebase\b`)
	codebaseSymbolRE      = regexp.MustCompile(`[A-Z][a-zA-Z0-9]{3,}`)
	codebaseIdentifierRE  = regexp.MustCompile(`[A-Z][a-z0-9]*(?:[A-Z][a-zA-Z0-9]+)+`)
)

// MergeCodebaseAttachments resolves @codebase in message content into prompt_attachments
// so non-desktop clients (DM, Slack, scenario harness) get the same semantic chunks as the IDE.
func MergeCodebaseAttachments(msg *protocol.Message) {
	if msg == nil || !codebaseMentionRE.MatchString(msg.Content) {
		return
	}
	plan := routing.PlanKnowledgeRoute(msg.Content)
	_ = MergeCodebaseForRoute(msg, plan)
}

// MergeCodebaseForRoute runs codeindex search when the knowledge plan includes codebase
// or the user explicitly mentions @codebase.
func MergeCodebaseForRoute(msg *protocol.Message, plan routing.KnowledgePlan) bool {
	if msg == nil {
		return false
	}
	explicit := codebaseMentionRE.MatchString(msg.Content)
	// Respect context_scope=none unless the user explicitly asked @codebase.
	if !explicit && ResolveContextScope(msg) == ContextScopeNone {
		return false
	}
	if !explicit && !ShouldRunCodebaseSearch(plan) {
		return false
	}
	repoPath := workspacePathFromMetadata(msg)
	if repoPath == "" {
		return false
	}
	query := strings.TrimSpace(codebaseMentionRE.ReplaceAllString(msg.Content, ""))
	if query == "" {
		query = strings.TrimSpace(msg.Content)
	}
	if query == "" {
		return false
	}
	paths := scopedRepoPathsFromMetadata(msg)
	if len(paths) == 0 {
		paths = []string{repoPath}
	}
	return mergeCodebaseSearchIntoMessage(msg, paths, query)
}

func mergeCodebaseSearchIntoMessage(msg *protocol.Message, repoPaths []string, query string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, repoPath := range repoPaths {
		meta, _ := codeindex.Status(repoPath)
		if !meta.Ready && !meta.Building {
			codeindex.BuildIndexAsync(repoPath)
		}
	}
	symbols := codebaseIdentifierRE.FindAllString(query, -1)
	if len(symbols) == 0 {
		symbols = codebaseSymbolRE.FindAllString(query, -1)
	}
	hits, err := codeintel.SemanticSearchMulti(ctx, repoPaths, query, 4, 12)
	semanticOK := err == nil && len(hits) > 0
	if semanticOK && len(symbols) > 0 && !semanticHitsContainAnySymbol(hits, symbols) {
		// Semantic chunks can miss exact identifiers (e.g. ComputeObscureWidget in a deep path).
		semanticOK = false
	}
	if !semanticOK {
		return mergeCodebaseSearchFallback(msg, repoPaths, query)
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	existing := promptAttachmentsSlice(msg.Metadata[MetadataPromptAttachments])
	for _, h := range hits {
		existing = append(existing, map[string]interface{}{
			"type":      "codebase_chunk",
			"path":      h.Path,
			"content":   h.Content,
			"repo_path": h.RepoPath,
			"repo_name": h.RepoName,
		})
	}
	msg.Metadata[MetadataPromptAttachments] = existing
	if msg.Metadata["injected_codebase_count"] == nil {
		msg.Metadata["injected_codebase_count"] = len(hits)
	}
	msg.Metadata["codebase_answer_from_attachments"] = true
	return true
}

func semanticHitsContainAnySymbol(hits []codeintel.RepoSearchHit, symbols []string) bool {
	if len(hits) == 0 || len(symbols) == 0 {
		return false
	}
	for _, sym := range symbols {
		for _, h := range hits {
			if strings.Contains(h.Content, sym) || strings.Contains(h.Path, sym) {
				return true
			}
		}
	}
	return false
}

func mergeCodebaseSearchFallback(msg *protocol.Message, repoPaths []string, query string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var results []codeindex.SearchResult
	var resultRepos []string
	seen := make(map[string]bool)
	appendResult := func(repoPath string, r codeindex.SearchResult) {
		prefix := r.Content
		if len(prefix) > 80 {
			prefix = prefix[:80]
		}
		key := repoPath + "\x00" + r.Path + "\x00" + prefix
		if seen[key] {
			return
		}
		seen[key] = true
		results = append(results, r)
		resultRepos = append(resultRepos, repoPath)
	}

	symbols := codebaseIdentifierRE.FindAllString(query, -1)
	if len(symbols) == 0 {
		symbols = codebaseSymbolRE.FindAllString(query, -1)
	}
	for _, repoPath := range repoPaths {
		for _, sym := range symbols {
			for _, r := range grepWorkspaceSymbol(repoPath, sym, 4) {
				appendResult(repoPath, r)
				if len(results) >= 12 {
					break
				}
			}
			if len(results) >= 12 {
				break
			}
		}
		if len(results) >= 12 {
			break
		}
	}

	for _, repoPath := range repoPaths {
		for _, q := range codebaseSearchQueries(query) {
			part, err := codeindex.Search(ctx, repoPath, q, 4)
			if err != nil {
				continue
			}
			for _, r := range part {
				appendResult(repoPath, r)
				if len(results) >= 12 {
					break
				}
			}
			if len(results) >= 12 {
				break
			}
		}
		if len(results) >= 12 {
			break
		}
	}
	if len(results) == 0 {
		return false
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	existing := promptAttachmentsSlice(msg.Metadata[MetadataPromptAttachments])
	for i, r := range results {
		repoPath := resultRepos[i]
		existing = append(existing, map[string]interface{}{
			"type":      "codebase_chunk",
			"path":      r.Path,
			"content":   r.Content,
			"repo_path": repoPath,
			"repo_name": codeintelRepoName(repoPath),
		})
	}
	msg.Metadata[MetadataPromptAttachments] = existing
	if msg.Metadata["injected_codebase_count"] == nil {
		msg.Metadata["injected_codebase_count"] = len(results)
	}
	msg.Metadata["codebase_answer_from_attachments"] = true
	return true
}

func codeintelRepoName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "repo"
	}
	path = strings.TrimRight(path, "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func scopedRepoPathsFromMetadata(msg *protocol.Message) []string {
	scoped := scopedWorkspacesFromMetadata(msg)
	if len(scoped) == 0 {
		return nil
	}
	out := make([]string, 0, len(scoped))
	for _, ref := range scoped {
		if ref.Path != "" {
			out = append(out, ref.Path)
		}
	}
	return out
}

func workspacePathFromMetadata(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok || raw == nil {
		return ""
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	p, _ := m["workspace_path"].(string)
	return strings.TrimSpace(p)
}

var codebaseSourceExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".py": true, ".rs": true, ".java": true, ".css": true, ".md": true,
}

func grepWorkspaceSymbol(repoPath, symbol string, limit int) []codeindex.SearchResult {
	if symbol == "" || limit <= 0 {
		return nil
	}
	var out []codeindex.SearchResult
	_ = filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", "dist", "target", "build", ".neural-junkie":
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !codebaseSourceExts[filepath.Ext(path)] {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(b), symbol) {
			return nil
		}
		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		content := string(b)
		if len(content) > maxPerAttachmentContent {
			content = content[:maxPerAttachmentContent] + "\n…"
		}
		out = append(out, codeindex.SearchResult{Path: filepath.ToSlash(rel), Content: content})
		if len(out) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	return out
}

func codebaseSearchQueries(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	seen := map[string]bool{query: true}
	out := []string{query}
	for _, sym := range codebaseSymbolRE.FindAllString(query, -1) {
		if seen[sym] {
			continue
		}
		seen[sym] = true
		out = append(out, sym)
	}
	return out
}

func promptAttachmentsSlice(raw interface{}) []interface{} {
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]interface{}, 0, len(arr))
	out = append(out, arr...)
	return out
}
