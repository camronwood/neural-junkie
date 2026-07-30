package codeindex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex/graph"
	"github.com/camronwood/neural-junkie/internal/codeindex/store"
	"github.com/camronwood/neural-junkie/internal/embed"
	"github.com/camronwood/neural-junkie/internal/git"
	"github.com/camronwood/neural-junkie/internal/repo"
	"github.com/camronwood/neural-junkie/internal/workspacefiles"
)

const (
	defaultChunkLines = 100
	maxChunkContent   = 4000
	maxFilesPerBuild  = 50000
)

var (
	globalMu     sync.RWMutex
	globalClient *embed.Client
	globalModel  = embed.DefaultModel
	buildMu      sync.Mutex
	building     = map[string]bool{}
)

// SetEmbedClient configures the Ollama embed client for codebase search.
func SetEmbedClient(c *embed.Client) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalClient = c
}

func getClient() *embed.Client {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalClient
}

// RepoHash returns a stable hash for a repo path.
func RepoHash(repoPath string) string {
	abs, _ := filepath.Abs(filepath.Clean(repoPath))
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:8])
}

func indexDir(repoPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "codeindex", RepoHash(repoPath)), nil
}

func metaPath(repoPath string) (string, error) {
	dir, err := indexDir(repoPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "meta.json"), nil
}

func removeLegacyJSON(dir string) {
	_ = os.Remove(filepath.Join(dir, "chunks.json"))
	_ = os.Remove(filepath.Join(dir, "vectors.json"))
	_ = os.Remove(filepath.Join(dir, "vectors.db"))
}

// Status returns index metadata for a repo.
// Legacy schema (< CurrentSchemaVersion) or missing index.db is reported as not ready.
func Status(repoPath string) (IndexMeta, error) {
	meta := IndexMeta{RepoPath: repoPath, RepoHash: RepoHash(repoPath), EmbeddingModel: globalModel}
	mp, err := metaPath(repoPath)
	if err != nil {
		return meta, err
	}
	raw, err := os.ReadFile(mp)
	if err != nil {
		if os.IsNotExist(err) {
			return meta, nil
		}
		return meta, err
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, err
	}
	dir, _ := indexDir(repoPath)
	if meta.SchemaVersion < CurrentSchemaVersion || !store.Exists(dir) {
		meta.Ready = false
	}
	buildMu.Lock()
	meta.Building = building[RepoHash(repoPath)]
	buildMu.Unlock()
	return meta, nil
}

// Search finds relevant code chunks using hybrid embed + keyword retrieval.
// Candidates come from SQLite FTS/LIKE — never a full chunks.json load.
func Search(ctx context.Context, repoPath, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	repoPath = strings.TrimSpace(repoPath)
	query = strings.TrimSpace(query)
	if repoPath == "" || query == "" {
		return nil, fmt.Errorf("repo_path and query required")
	}

	meta, _ := Status(repoPath)
	if !meta.Ready {
		return keywordSearch(ctx, repoPath, query, limit)
	}

	dir, err := indexDir(repoPath)
	if err != nil {
		return keywordSearch(ctx, repoPath, query, limit)
	}
	sqlStore, err := store.Open(dir)
	if err != nil {
		return keywordSearch(ctx, repoPath, query, limit)
	}
	defer sqlStore.Close()

	candidates := sqlStore.LexicalCandidates(query, limit*10)
	if len(candidates) == 0 {
		return keywordSearch(ctx, repoPath, query, limit)
	}

	ids := make([]string, len(candidates))
	for i, ch := range candidates {
		ids[i] = ch.ID
	}
	vectors := sqlStore.GetVectors(ids)

	client := getClient()
	var queryVec []float64
	embedOK := false
	if client != nil {
		queryVec, err = client.Embed(ctx, query, false)
		embedOK = err == nil && len(queryVec) > 0
	}

	scored := make([]embed.ScoredItem[SearchResult], 0, len(candidates))
	for i, ch := range candidates {
		score := embed.KeywordScore(query, ch.Path+" "+ch.Content)
		if embedOK {
			if vec, ok := vectors[ch.ID]; ok && len(vec) > 0 {
				score = embed.CosineSimilarity(queryVec, vec)
			}
		}
		if score <= 0 {
			// FTS/LIKE already selected these rows; keep a rank floor so punctuation
			// quirks in KeywordScore do not discard every candidate.
			score = 1 / (1 + 0.15*float64(i))
		}
		content := ch.Content
		if len(content) > maxChunkContent {
			content = content[:maxChunkContent] + "\n…"
		}
		scored = append(scored, embed.ScoredItem[SearchResult]{
			Item:  SearchResult{Path: ch.Path, Content: content, Score: score},
			Score: score,
		})
	}

	return embed.TopKByScore(scored, limit), nil
}

func keywordSearch(ctx context.Context, repoPath, query string, limit int) ([]SearchResult, error) {
	paths, err := workspacefiles.Search(ctx, repoPath, query, limit*3)
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
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
		if !IsReadableSourceFile(full, rel) {
			continue
		}
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(b)
		if len(content) > maxChunkContent {
			content = content[:maxChunkContent] + "\n…"
		}
		results = append(results, SearchResult{Path: rel, Content: content})
	}
	return results, nil
}

// BuildIndex chunks and embeds files under repoPath (incremental when meta matches git HEAD).
func BuildIndex(ctx context.Context, repoPath string) error {
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

	head := ""
	if git.IsRepo(repoPath) {
		head, _ = git.RevParseHEAD(repoPath)
	}

	prevMeta, _ := Status(repoPath)
	if prevMeta.Ready && prevMeta.SchemaVersion >= CurrentSchemaVersion && prevMeta.GitHEAD == head && prevMeta.ChunkCount > 0 {
		return nil
	}

	files, err := listSourceFiles(ctx, repoPath)
	if err != nil {
		return err
	}
	if len(files) > maxFilesPerBuild {
		files = files[:maxFilesPerBuild]
	}

	var chunks []Chunk
	for _, rel := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !IsIndexableRelPath(rel) {
			continue
		}
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
		if !IsReadableSourceFile(full, rel) {
			continue
		}
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		if LooksLikeBinary(b) {
			continue
		}
		chunks = append(chunks, chunkFileAST(rel, string(b))...)
	}

	sqlStore, err := store.Open(dir)
	if err != nil {
		return err
	}
	defer sqlStore.Close()

	recs := make([]store.ChunkRecord, len(chunks))
	keep := make(map[string]struct{}, len(chunks))
	for i, ch := range chunks {
		recs[i] = store.ChunkRecord{
			ID: ch.ID, Path: ch.Path, Start: ch.Start, End: ch.End, Content: ch.Content,
		}
		keep[ch.ID] = struct{}{}
	}
	if err := sqlStore.ReplaceAllChunks(recs); err != nil {
		return err
	}

	client := getClient()
	model := globalModel
	if client != nil {
		model = client.Model
		for _, ch := range chunks {
			if ctx.Err() != nil {
				break
			}
			if _, ok := sqlStore.Get(ch.ID); ok {
				continue
			}
			vec, err := client.Embed(ctx, ch.Path+"\n"+ch.Content, true)
			if err != nil {
				continue
			}
			_ = sqlStore.Put(ch.ID, vec)
		}
	}
	_ = sqlStore.DeleteMissing(keep)

	removeLegacyJSON(dir)

	meta := IndexMeta{
		RepoPath:       repoPath,
		RepoHash:       hash,
		ChunkCount:     len(chunks),
		EmbeddingModel: model,
		LastBuiltAt:    time.Now().UTC(),
		GitHEAD:        head,
		SchemaVersion:  CurrentSchemaVersion,
		Ready:          len(chunks) > 0,
	}
	mp, _ := metaPath(repoPath)
	mb, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(mp, mb, 0o644); err != nil {
		return err
	}
	graph.BuildIndexAsync(repoPath)
	return nil
}

// BuildIndexAsync starts a background index build.
func BuildIndexAsync(repoPath string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_ = BuildIndex(ctx, repoPath)
	}()
}

func listSourceFiles(ctx context.Context, repoPath string) ([]string, error) {
	storage, err := repo.NewStorage()
	if err == nil {
		cacheKey, kerr := storage.GetCacheKeyForPath(repoPath)
		if kerr == nil && storage.IndexExists(cacheKey) {
			idx, lerr := storage.LoadIndex(cacheKey)
			if lerr == nil && idx != nil && len(idx.SourceFiles) > 0 {
				paths := make([]string, 0, len(idx.SourceFiles))
				for p := range idx.SourceFiles {
					paths = append(paths, p)
				}
				return filterIndexablePaths(paths), nil
			}
		}
	}
	paths, err := workspacefiles.Search(ctx, repoPath, "", maxFilesPerBuild)
	if err != nil {
		return nil, err
	}
	return filterIndexablePaths(paths), nil
}

func chunkFile(relPath, content string) []Chunk {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil
	}
	var chunks []Chunk
	for start := 0; start < len(lines); start += defaultChunkLines {
		end := start + defaultChunkLines
		if end > len(lines) {
			end = len(lines)
		}
		body := strings.Join(lines[start:end], "\n")
		if strings.TrimSpace(body) == "" {
			continue
		}
		id := fmt.Sprintf("%s:%d-%d", relPath, start+1, end)
		chunks = append(chunks, Chunk{
			ID:      id,
			Path:    relPath,
			Start:   start + 1,
			End:     end,
			Content: body,
		})
		if end >= len(lines) {
			break
		}
	}
	return chunks
}
