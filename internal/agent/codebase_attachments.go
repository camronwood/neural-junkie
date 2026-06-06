package agent

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	codebaseMentionRE  = regexp.MustCompile(`(?i)@codebase\b`)
	codebaseSymbolRE   = regexp.MustCompile(`[A-Z][a-zA-Z0-9]{3,}`)
)

// MergeCodebaseAttachments resolves @codebase in message content into prompt_attachments
// so non-desktop clients (DM, Slack, scenario harness) get the same semantic chunks as the IDE.
func MergeCodebaseAttachments(msg *protocol.Message) {
	if msg == nil || !codebaseMentionRE.MatchString(msg.Content) {
		return
	}
	repoPath := workspacePathFromMetadata(msg)
	if repoPath == "" {
		return
	}
	query := strings.TrimSpace(codebaseMentionRE.ReplaceAllString(msg.Content, ""))
	if query == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	meta, _ := codeindex.Status(repoPath)
	if !meta.Ready && !meta.Building {
		codeindex.BuildIndexAsync(repoPath)
	}
	var results []codeindex.SearchResult
	seen := make(map[string]bool)
	for _, q := range codebaseSearchQueries(query) {
		part, err := codeindex.Search(ctx, repoPath, q, 8)
		if err != nil {
			continue
		}
		for _, r := range part {
			prefix := r.Content
			if len(prefix) > 80 {
				prefix = prefix[:80]
			}
			key := r.Path + "\x00" + prefix
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, r)
			if len(results) >= 8 {
				break
			}
		}
		if len(results) >= 8 {
			break
		}
	}
	if len(results) == 0 {
		for _, sym := range codebaseSymbolRE.FindAllString(query, -1) {
			results = append(results, grepWorkspaceSymbol(repoPath, sym, 4)...)
			if len(results) >= 8 {
				break
			}
		}
	}
	if len(results) == 0 {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	existing := promptAttachmentsSlice(msg.Metadata[MetadataPromptAttachments])
	for _, r := range results {
		existing = append(existing, map[string]interface{}{
			"type":    "codebase_chunk",
			"path":    r.Path,
			"content": r.Content,
		})
	}
	msg.Metadata[MetadataPromptAttachments] = existing
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
