package codeindex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex/store"
	"github.com/camronwood/neural-junkie/internal/repo"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
	"github.com/camronwood/neural-junkie/internal/workspacefiles"
)

// BuildIndexViaBackend indexes source files read through workspace backend.
func BuildIndexViaBackend(ctx context.Context, repoPath string, b workspacebackend.Backend) error {
	if b == nil {
		return BuildIndex(ctx, repoPath)
	}
	repoPath = filepath.Clean(repoPath)
	hash := RepoHash(repoPath)
	buildMu.Lock()
	if building[hash] {
		buildMu.Unlock()
		return nil
	}
	building[hash] = true
	buildMu.Unlock()
	defer func() {
		buildMu.Lock()
		delete(building, hash)
		buildMu.Unlock()
	}()

	dir, err := indexDir(repoPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	files, err := workspacebackend.ListFilesRecursive(ctx, b, ".", maxFilesPerBuild)
	if err != nil {
		return err
	}
	var chunks []Chunk
	for _, rel := range files {
		if ctx.Err() != nil {
			break
		}
		rel = filepath.ToSlash(rel)
		if !IsIndexableRelPath(rel) {
			continue
		}
		data, err := b.ReadFile(ctx, rel)
		if err != nil {
			continue
		}
		if int64(len(data)) > repo.MaxFileSize || LooksLikeBinary(data) {
			continue
		}
		chunks = append(chunks, chunkFile(rel, string(data))...)
	}

	sqlStore, err := store.Open(dir)
	if err != nil {
		return err
	}
	defer sqlStore.Close()

	recs := make([]store.ChunkRecord, len(chunks))
	for i, ch := range chunks {
		recs[i] = store.ChunkRecord{
			ID: ch.ID, Path: ch.Path, Start: ch.Start, End: ch.End, Content: ch.Content,
		}
	}
	if err := sqlStore.ReplaceAllChunks(recs); err != nil {
		return err
	}
	removeLegacyJSON(dir)

	meta := IndexMeta{
		RepoPath:       repoPath,
		RepoHash:       hash,
		ChunkCount:     len(chunks),
		EmbeddingModel: globalModel,
		LastBuiltAt:    time.Now().UTC(),
		SchemaVersion:  CurrentSchemaVersion,
		Ready:          len(chunks) > 0,
	}
	mp, _ := metaPath(repoPath)
	mb, _ := json.MarshalIndent(meta, "", "  ")
	return os.WriteFile(mp, mb, 0o644)
}

// SearchViaBackend searches codebase using backend for keyword fallback.
func SearchViaBackend(ctx context.Context, repoPath string, b workspacebackend.Backend, query string, limit int) ([]SearchResult, error) {
	results, err := Search(ctx, repoPath, query, limit)
	if err == nil && len(results) > 0 {
		return results, nil
	}
	if b == nil {
		return results, err
	}
	return keywordSearchBackend(ctx, b, query, limit)
}

func keywordSearchBackend(ctx context.Context, b workspacebackend.Backend, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	paths, err := workspacefiles.SearchBackend(ctx, b, query, limit*3)
	if err != nil {
		return nil, err
	}
	var results []SearchResult
	for _, rel := range paths {
		if len(results) >= limit {
			break
		}
		if !IsIndexableRelPath(rel) {
			continue
		}
		data, err := b.ReadFile(ctx, rel)
		if err != nil {
			continue
		}
		if LooksLikeBinary(data) {
			continue
		}
		content := string(data)
		if len(content) > maxChunkContent {
			content = content[:maxChunkContent] + "\n…"
		}
		results = append(results, SearchResult{Path: rel, Content: content})
	}
	return results, nil
}
