// Package embed provides Ollama embedding and vector utilities shared across features.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultModel = "nomic-embed-text"

// Record is a cached embedding vector.
type Record struct {
	Model      string    `json:"model"`
	Vector     []float64 `json:"vector"`
	EmbeddedAt time.Time `json:"embedded_at"`
}

type embedFile struct {
	Entries map[string]Record `json:"entries"`
}

// Store caches vectors in a JSON sidecar file.
type Store struct {
	mu   sync.RWMutex
	path string
	data embedFile
}

// NewStore opens or creates an embed store at path (empty = ~/.neural-junkie/learning-embeddings.json).
func NewStore(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			path = "learning-embeddings.json"
		} else {
			path = filepath.Join(home, ".neural-junkie", "learning-embeddings.json")
		}
	}
	es := &Store{path: path, data: embedFile{Entries: map[string]Record{}}}
	if err := es.load(); err != nil {
		return nil, err
	}
	return es, nil
}

func (es *Store) load() error {
	es.mu.Lock()
	defer es.mu.Unlock()
	raw, err := os.ReadFile(es.path)
	if err != nil {
		if os.IsNotExist(err) {
			es.data.Entries = map[string]Record{}
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		es.data.Entries = map[string]Record{}
		return nil
	}
	return json.Unmarshal(raw, &es.data)
}

func (es *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(es.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(es.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(es.path, raw, 0o644)
}

func (es *Store) Get(id string) (Record, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	r, ok := es.data.Entries[id]
	return r, ok
}

func (es *Store) Set(id, model string, vec []float64) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.data.Entries == nil {
		es.data.Entries = map[string]Record{}
	}
	es.data.Entries[id] = Record{Model: model, Vector: vec, EmbeddedAt: time.Now().UTC()}
	return es.saveLocked()
}

func (es *Store) Delete(id string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.data.Entries, id)
	_ = es.saveLocked()
}

func (es *Store) Count() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.data.Entries)
}

// Client calls Ollama /api/embed.
type Client struct {
	Endpoint string
	Model    string
	FastHTTP *http.Client
	BatchHTTP *http.Client
}

func NewClient(endpoint, model string) *Client {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		Endpoint:  strings.TrimRight(endpoint, "/"),
		Model:     model,
		FastHTTP:  &http.Client{Timeout: 200 * time.Millisecond},
		BatchHTTP: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Embed(ctx context.Context, text string, batch bool) ([]float64, error) {
	httpClient := c.FastHTTP
	if batch {
		httpClient = c.BatchHTTP
	}
	body, _ := json.Marshal(map[string]string{"model": c.Model, "prompt": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed status %d", resp.StatusCode)
	}
	var out struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Embedding, nil
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func KeywordScore(query, content string) float64 {
	if query == "" {
		return 0
	}
	qTokens := tokenSet(query)
	if len(qTokens) == 0 {
		return 0
	}
	cTokens := tokenSet(content)
	if len(cTokens) == 0 {
		return 0
	}
	hits := 0
	for t := range qTokens {
		if cTokens[t] {
			hits++
		}
	}
	return float64(hits) / float64(len(qTokens))
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range strings.Fields(strings.ToLower(s)) {
		if len(t) > 1 {
			out[t] = true
		}
	}
	return out
}

// ScoredItem pairs an item with a relevance score.
type ScoredItem[T any] struct {
	Item  T
	Score float64
}

// TopKByScore returns the top k items by score descending.
func TopKByScore[T any](scored []ScoredItem[T], k int) []T {
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if k <= 0 || k > len(scored) {
		k = len(scored)
	}
	out := make([]T, k)
	for i := 0; i < k; i++ {
		out[i] = scored[i].Item
	}
	return out
}
