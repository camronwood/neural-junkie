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

func chunksPath(repoPath string) (string, error) {
	dir, err := indexDir(repoPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "chunks.json"), nil
}

func vectorsPath(repoPath string) (string, error) {
	dir, err := indexDir(repoPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vectors.json"), nil
}

func metaPath(repoPath string) (string, error) {
	dir, err := indexDir(repoPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "meta.json"), nil
}

// Status returns index metadata for a repo.
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
	buildMu.Lock()
	meta.Building = building[RepoHash(repoPath)]
	buildMu.Unlock()
	return meta, nil
}

// Search finds relevant code chunks using hybrid embed + keyword retrieval.
func Search(ctx context.Context, repoPath, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	repoPath = strings.TrimSpace(repoPath)
	query = strings.TrimSpace(query)
	if repoPath == "" || query == "" {
		return nil, fmt.Errorf("repo_path and query required")
	}

	chunks, err := loadChunks(repoPath)
	if err != nil || len(chunks) == 0 {
		return keywordSearch(ctx, repoPath, query, limit)
	}

	vecStore, _ := embed.NewStore("")
	vp, _ := vectorsPath(repoPath)
	if vp != "" {
		vecStore, _ = embed.NewStore(vp)
	}

	client := getClient()
	var queryVec []float64
	embedOK := false
	if client != nil {
		queryVec, err = client.Embed(ctx, query, false)
		embedOK = err == nil && len(queryVec) > 0
	}

	candidates := prefilterChunks(chunks, query, limit*10)
	if len(candidates) == 0 {
		candidates = chunks
		if len(candidates) > limit*10 {
			candidates = candidates[:limit*10]
		}
	}

	scored := make([]embed.ScoredItem[SearchResult], 0, len(candidates))
	for _, ch := range candidates {
		score := embed.KeywordScore(query, ch.Path+" "+ch.Content)
		if embedOK && vecStore != nil {
			if rec, ok := vecStore.Get(ch.ID); ok && len(rec.Vector) > 0 {
				score = embed.CosineSimilarity(queryVec, rec.Vector)
			}
		}
		if score <= 0 {
			continue
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

	if len(scored) == 0 {
		return keywordSearch(ctx, repoPath, query, limit)
	}
	top := embed.TopKByScore(scored, limit)
	return top, nil
}

func prefilterChunks(chunks []Chunk, query string, max int) []Chunk {
	q := strings.ToLower(query)
	var out []Chunk
	for _, ch := range chunks {
		if strings.Contains(strings.ToLower(ch.Path+" "+ch.Content), q) ||
			embed.KeywordScore(query, ch.Path+" "+ch.Content) > 0.1 {
			out = append(out, ch)
			if len(out) >= max {
				break
			}
		}
	}
	return out
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
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
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

func loadChunks(repoPath string) ([]Chunk, error) {
	cp, err := chunksPath(repoPath)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(cp)
	if err != nil {
		return nil, err
	}
	var chunks []Chunk
	if err := json.Unmarshal(raw, &chunks); err != nil {
		return nil, err
	}
	return chunks, nil
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

	existing, _ := loadChunks(repoPath)
	prevMeta, _ := Status(repoPath)
	if prevMeta.Ready && prevMeta.GitHEAD == head && len(existing) > 0 {
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
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		chunks = append(chunks, chunkFileAST(rel, string(b))...)
	}

	cp, _ := chunksPath(repoPath)
	raw, _ := json.MarshalIndent(chunks, "", "  ")
	if err := os.WriteFile(cp, raw, 0o644); err != nil {
		return err
	}

	vp, _ := vectorsPath(repoPath)
	vecStore, err := embed.NewStore(vp)
	if err != nil {
		return err
	}
	client := getClient()
	model := globalModel
	if client != nil {
		model = client.Model
		for _, ch := range chunks {
			if _, ok := vecStore.Get(ch.ID); ok {
				continue
			}
			if client == nil {
				break
			}
			vec, err := client.Embed(ctx, ch.Path+"\n"+ch.Content, true)
			if err != nil {
				continue
			}
			_ = vecStore.Set(ch.ID, model, vec)
			if sqlStore, err := store.Open(dir); err == nil {
				_ = sqlStore.Put(ch.ID, vec)
				_ = sqlStore.Close()
			}
		}
	}
	keep := make(map[string]struct{}, len(chunks))
	for _, ch := range chunks {
		keep[ch.ID] = struct{}{}
	}
	if sqlStore, err := store.Open(dir); err == nil {
		_ = sqlStore.DeleteMissing(keep)
		_ = sqlStore.Close()
	}

	meta := IndexMeta{
		RepoPath:       repoPath,
		RepoHash:       hash,
		ChunkCount:     len(chunks),
		EmbeddingModel: model,
		LastBuiltAt:    time.Now().UTC(),
		GitHEAD:        head,
		Ready:          len(chunks) > 0,
	}
	mp, _ := metaPath(repoPath)
	mb, _ := json.MarshalIndent(meta, "", "  ")
	return os.WriteFile(mp, mb, 0o644)
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
				return paths, nil
			}
		}
	}
	return workspacefiles.Search(ctx, repoPath, "", maxFilesPerBuild)
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
