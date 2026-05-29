package hfhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const hfModelsAPI = "https://huggingface.co/api/models"

// SearchHit is one model from Hugging Face Hub search.
type SearchHit struct {
	RepoID      string   `json:"repo_id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Downloads   int64    `json:"downloads,omitempty"`
	Likes       int64    `json:"likes,omitempty"`
	Modes       []string `json:"modes,omitempty"`
	Files       []CatalogFile `json:"files,omitempty"`
}

// SearchResult is returned by HF model search.
type SearchResult struct {
	Query   string      `json:"query"`
	Mode    string      `json:"mode"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	Total   int         `json:"total,omitempty"`
	Models  []SearchHit `json:"models"`
	HasMore bool        `json:"has_more"`
}

type hfSearchClient struct {
	httpClient *http.Client
	cacheMu    sync.RWMutex
	cache      map[string]cachedHFSearch
}

type cachedHFSearch struct {
	at     time.Time
	result SearchResult
}

var defaultHFSearch = &hfSearchClient{
	httpClient: &http.Client{Timeout: 25 * time.Second},
	cache:      make(map[string]cachedHFSearch),
}

const hfSearchCacheTTL = 3 * time.Minute

// SearchModels queries Hugging Face Hub. mode is "hosted" or "local".
func SearchModels(ctx context.Context, query, mode string, limit, offset int) (SearchResult, error) {
	return defaultHFSearch.search(ctx, query, mode, limit, offset)
}

// ListRepoGGUF returns .gguf filenames in a Hub repo (best-effort).
func ListRepoGGUF(ctx context.Context, repoID, token string) ([]CatalogFile, error) {
	return defaultHFSearch.listRepoFiles(ctx, repoID, token, "gguf")
}

// ListRepoAdapterFiles returns LoRA adapter safetensors in a Hub repo (best-effort).
func ListRepoAdapterFiles(ctx context.Context, repoID, token string) ([]CatalogFile, error) {
	return defaultHFSearch.listRepoFiles(ctx, repoID, token, "adapter")
}

func (c *hfSearchClient) listRepoFiles(ctx context.Context, repoID, token, fileKind string) ([]CatalogFile, error) {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return nil, fmt.Errorf("repo_id is required")
	}
	if entry, err := FindCatalogEntry(repoID); err == nil && len(entry.Files) > 0 {
		if fileKind == "adapter" && !IsAdapterEntry(entry) {
			return nil, fmt.Errorf("repo_id %q is not an adapter catalog entry", repoID)
		}
		if fileKind == "gguf" && IsAdapterEntry(entry) {
			return nil, fmt.Errorf("repo_id %q is an adapter entry, not GGUF", repoID)
		}
		return append([]CatalogFile(nil), entry.Files...), nil
	}

	target := fmt.Sprintf("https://huggingface.co/api/models/%s/tree/main?recursive=true", url.PathEscape(repoID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Neural-Junkie/1.0")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hf list files: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("access denied (%d): accept the model license on Hugging Face and set HF_TOKEN for gated repos", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("hf list files: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var nodes []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int64  `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("hf list files decode: %w", err)
	}

	var files []CatalogFile
	for _, node := range nodes {
		if node.Type != "file" {
			continue
		}
		lower := strings.ToLower(node.Path)
		switch fileKind {
		case "adapter":
			if !strings.HasSuffix(lower, ".safetensors") {
				continue
			}
			if node.Size > 500*1024*1024 {
				continue
			}
			if !isLikelyAdapterPath(lower) {
				continue
			}
		default:
			if !strings.HasSuffix(lower, ".gguf") {
				continue
			}
		}
		base := path.Base(node.Path)
		files = append(files, CatalogFile{
			Filename: base,
			Quant:    quantFromFilename(base),
		})
	}
	if len(files) == 0 {
		if fileKind == "adapter" {
			return nil, fmt.Errorf("no LoRA adapter files found in %s", repoID)
		}
		return nil, fmt.Errorf("no GGUF files found in %s", repoID)
	}
	if fileKind != "adapter" {
		files = preferCommonQuants(files)
	}
	if len(files) > 12 {
		files = files[:12]
	}
	return files, nil
}

func isLikelyAdapterPath(pathLower string) bool {
	if strings.Contains(pathLower, "adapter") {
		return true
	}
	base := path.Base(pathLower)
	return base == "adapter_model.safetensors" || strings.HasPrefix(base, "lora")
}

func (c *hfSearchClient) search(ctx context.Context, query, mode string, limit, offset int) (SearchResult, error) {
	query = strings.TrimSpace(query)
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "hosted"
	}
	if limit <= 0 {
		limit = 24
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	cacheKey := fmt.Sprintf("%s\x00%s\x00%d\x00%d", query, mode, limit, offset)
	c.cacheMu.RLock()
	if hit, ok := c.cache[cacheKey]; ok && time.Since(hit.at) < hfSearchCacheTTL {
		c.cacheMu.RUnlock()
		return hit.result, nil
	}
	c.cacheMu.RUnlock()

	params := url.Values{}
	if query != "" {
		params.Set("search", query)
	}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("full", "true")
	switch mode {
	case "local":
		params.Set("filter", "text-generation")
		params.Set("tags", "gguf")
	default:
		params.Set("filter", "text-generation")
		params.Set("inference", "warm")
	}

	target := hfModelsAPI + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return SearchResult{}, err
	}
	req.Header.Set("User-Agent", "Neural-Junkie/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResult{}, fmt.Errorf("hf search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return SearchResult{}, fmt.Errorf("hf search: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rows []struct {
		ID          string   `json:"id"`
		ModelID     string   `json:"modelId"`
		Tags        []string `json:"tags"`
		Downloads   int64    `json:"downloads"`
		Likes       int64    `json:"likes"`
		PipelineTag string   `json:"pipeline_tag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return SearchResult{}, fmt.Errorf("hf search decode: %w", err)
	}

	out := SearchResult{
		Query:   query,
		Mode:    mode,
		Limit:   limit,
		Offset:  offset,
		Models:  make([]SearchHit, 0, len(rows)),
		HasMore: len(rows) >= limit,
	}
	for _, row := range rows {
		repoID := strings.TrimSpace(row.ID)
		if repoID == "" {
			repoID = strings.TrimSpace(row.ModelID)
		}
		if repoID == "" {
			continue
		}
		hit := SearchHit{
			RepoID:    repoID,
			Title:     titleFromRepoID(repoID),
			Tags:      append([]string(nil), row.Tags...),
			Downloads: row.Downloads,
			Likes:     row.Likes,
		}
		if mode == "local" {
			hit.Modes = []string{"local"}
		} else {
			hit.Modes = []string{"hosted"}
		}
		if curated, err := FindCatalogEntry(repoID); err == nil {
			hit.Title = curated.Title
			hit.Description = curated.Description
			hit.Tags = append(curated.Tags, hit.Tags...)
			hit.Modes = append([]string(nil), curated.Modes...)
			hit.Files = append([]CatalogFile(nil), curated.Files...)
		}
		out.Models = append(out.Models, hit)
	}

	c.cacheMu.Lock()
	c.cache[cacheKey] = cachedHFSearch{at: time.Now(), result: out}
	c.cacheMu.Unlock()
	return out, nil
}

func (c *hfSearchClient) listGGUF(ctx context.Context, repoID, token string) ([]CatalogFile, error) {
	return c.listRepoFiles(ctx, repoID, token, "gguf")
}

func titleFromRepoID(repoID string) string {
	parts := strings.Split(repoID, "/")
	name := parts[len(parts)-1]
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	return name
}

func quantFromFilename(filename string) string {
	upper := strings.ToUpper(filename)
	for _, q := range []string{"Q4_K_M", "Q5_K_M", "Q4_0", "Q5_0", "Q8_0", "Q3_K_M", "Q2_K"} {
		if strings.Contains(upper, q) {
			return q
		}
	}
	return ""
}

func preferCommonQuants(files []CatalogFile) []CatalogFile {
	if len(files) <= 1 {
		return files
	}
	preferred := []string{"Q4_K_M", "Q5_K_M", "Q4_0", "Q8_0"}
	for _, want := range preferred {
		for _, f := range files {
			if strings.EqualFold(f.Quant, want) {
				rest := make([]CatalogFile, 0, len(files)-1)
				for _, other := range files {
					if other.Filename != f.Filename {
						rest = append(rest, other)
					}
				}
				return append([]CatalogFile{f}, rest...)
			}
		}
	}
	return files
}

// ResolveDownloadTarget resolves repo and filename for hub downloads.
func ResolveDownloadTarget(repoID, filename string) (downloadRepoID, resolvedFilename string, err error) {
	repoID = strings.TrimSpace(repoID)
	filename = strings.TrimSpace(filename)
	if repoID == "" {
		return "", "", fmt.Errorf("repo_id is required")
	}
	if entry, err := FindCatalogEntry(repoID); err == nil {
		fn, err := ResolveDownloadFilename(entry, filename)
		if err != nil {
			return "", "", err
		}
		return ResolveDownloadRepoID(entry), fn, nil
	}
	if filename == "" {
		return "", "", fmt.Errorf("filename is required for repos outside the curated catalog")
	}
	if strings.Contains(filename, "..") {
		return "", "", fmt.Errorf("invalid filename")
	}
	clean := filepath.Clean(filename)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", "", fmt.Errorf("invalid filename")
	}
	return repoID, filepath.ToSlash(clean), nil
}

// MergeCatalogWithSearch prepends curated rows and appends unique search hits.
func MergeCatalogWithSearch(curated []LibraryModel, hits []SearchHit, mode string) []LibraryModel {
	out := append([]LibraryModel(nil), curated...)
	seen := make(map[string]struct{}, len(curated)+len(hits))
	for _, row := range curated {
		if !catalogMatchesMode(&row, mode) {
			continue
		}
		seen[row.RepoID] = struct{}{}
	}
	for _, hit := range hits {
		if _, ok := seen[hit.RepoID]; ok {
			continue
		}
		if !searchHitMatchesMode(&hit, mode) {
			continue
		}
		seen[hit.RepoID] = struct{}{}
		out = append(out, LibraryModel{
			RepoID:      hit.RepoID,
			Title:       hit.Title,
			Description: hit.Description,
			Tags:        hit.Tags,
			Modes:       hit.Modes,
			Files:       hit.Files,
		})
	}
	return out
}

func catalogMatchesMode(entry *LibraryModel, mode string) bool {
	for _, m := range entry.Modes {
		if m == mode {
			return true
		}
	}
	return false
}

func searchHitMatchesMode(hit *SearchHit, mode string) bool {
	for _, m := range hit.Modes {
		if m == mode {
			return true
		}
	}
	return mode == "hosted"
}
