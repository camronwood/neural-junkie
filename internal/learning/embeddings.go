package learning

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

type embedRecord struct {
	Model      string    `json:"model"`
	Vector     []float64 `json:"vector"`
	EmbeddedAt time.Time `json:"embedded_at"`
}

type embedFile struct {
	Entries map[string]embedRecord `json:"entries"`
}

// EmbedStore caches vectors in learning-embeddings.json.
type EmbedStore struct {
	mu   sync.RWMutex
	path string
	data embedFile
}

func DefaultEmbedPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "learning-embeddings.json"
	}
	return filepath.Join(home, ".neural-junkie", "learning-embeddings.json")
}

func NewEmbedStore(path string) (*EmbedStore, error) {
	if path == "" {
		path = DefaultEmbedPath()
	}
	es := &EmbedStore{path: path, data: embedFile{Entries: map[string]embedRecord{}}}
	if err := es.load(); err != nil {
		return nil, err
	}
	return es, nil
}

func (es *EmbedStore) load() error {
	es.mu.Lock()
	defer es.mu.Unlock()
	raw, err := os.ReadFile(es.path)
	if err != nil {
		if os.IsNotExist(err) {
			es.data.Entries = map[string]embedRecord{}
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		es.data.Entries = map[string]embedRecord{}
		return nil
	}
	return json.Unmarshal(raw, &es.data)
}

func (es *EmbedStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(es.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(es.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(es.path, raw, 0o644)
}

func (es *EmbedStore) Get(id string) (embedRecord, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	r, ok := es.data.Entries[id]
	return r, ok
}

func (es *EmbedStore) Set(id, model string, vec []float64) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.data.Entries == nil {
		es.data.Entries = map[string]embedRecord{}
	}
	es.data.Entries[id] = embedRecord{Model: model, Vector: vec, EmbeddedAt: time.Now().UTC()}
	return es.saveLocked()
}

func (es *EmbedStore) Delete(id string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.data.Entries, id)
	_ = es.saveLocked()
}

func (es *EmbedStore) Count() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.data.Entries)
}

var (
	globalEmbedStore *EmbedStore
	embedEndpoint    = "http://localhost:11434"
	embedModel       = DefaultEmbedModel
	embedHTTP        = &http.Client{Timeout: 200 * time.Millisecond}
	collabResolver   func(channel string) string
	onEntryChanged   func(entry Entry) // async re-embed hook
)

func SetEmbedStore(es *EmbedStore) { globalEmbedStore = es }

func SetEmbedConfig(endpoint, model string) {
	if endpoint != "" {
		embedEndpoint = strings.TrimRight(endpoint, "/")
	}
	if model != "" {
		embedModel = model
	}
}

func SetCollabResolver(fn func(channel string) string) { collabResolver = fn }

func SetOnEntryChanged(fn func(entry Entry)) { onEntryChanged = fn }

func ResolveCollabID(channel string) string {
	if collabResolver == nil {
		return ""
	}
	return collabResolver(channel)
}

func IndexReady() bool {
	return globalEmbedStore != nil && globalEmbedStore.Count() > 0
}

func ScheduleEmbed(entry Entry) {
	if onEntryChanged != nil {
		onEntryChanged(entry)
		return
	}
	go func(e Entry) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = ensureVector(ctx, e.ID, e.Content)
	}(entry)
}

type scoredEntry struct {
	entry Entry
	score float64
}

func SelectForPrompt(ctx context.Context, pctx PromptContext, agentID string) (global, agent, collab []Entry, ids []string) {
	if globalStore == nil || !learningEnabled() {
		return nil, nil, nil, nil
	}
	userID := pctx.UserID
	query := strings.TrimSpace(pctx.Query)
	collabID := pctx.CollaborationID
	if collabID == "" && pctx.Channel != "" {
		collabID = ResolveCollabID(pctx.Channel)
		pctx.CollaborationID = collabID
	}

	poolGlobal := globalStore.ListFiltered(Filter{UserID: userID, Scope: ScopeGlobal, IncludeLegacy: true})
	poolAgent := globalStore.ListFiltered(Filter{UserID: userID, Scope: ScopeAgent, AgentID: agentID, IncludeLegacy: true})
	poolCollab := []Entry{}
	if collabID != "" {
		poolCollab = globalStore.ListFiltered(Filter{UserID: userID, Scope: ScopeCollaboration, CollaborationID: collabID, IncludeLegacy: true})
	}

	queryVec, embedOK := embedQuery(ctx, query)
	global = topK(poolGlobal, query, queryVec, embedOK, DefaultGlobalTopK)
	agent = topK(poolAgent, query, queryVec, embedOK, DefaultAgentTopK)
	collab = topK(poolCollab, query, queryVec, embedOK, DefaultCollabTopK)

	for _, e := range global {
		ids = append(ids, e.ID)
	}
	for _, e := range agent {
		ids = append(ids, e.ID)
	}
	for _, e := range collab {
		ids = append(ids, e.ID)
	}
	if len(ids) > 0 {
		go globalStore.RecordUse(ids)
	}
	return global, agent, collab, ids
}

func topK(pool []Entry, query string, queryVec []float64, embedOK bool, k int) []Entry {
	if len(pool) == 0 {
		return nil
	}
	if k <= 0 {
		k = len(pool)
	}
	scored := make([]scoredEntry, 0, len(pool))
	for _, e := range pool {
		score := keywordScore(query, e.Content)
		if embedOK && globalEmbedStore != nil {
			if rec, ok := globalEmbedStore.Get(e.ID); ok && len(rec.Vector) > 0 {
				score = cosineSimilarity(queryVec, rec.Vector)
			}
		}
		scored = append(scored, scoredEntry{entry: e, score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	if len(scored) > k {
		scored = scored[:k]
	}
	out := make([]Entry, len(scored))
	for i, s := range scored {
		out[i] = s.entry
	}
	return out
}

func keywordScore(query, content string) float64 {
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

func cosineSimilarity(a, b []float64) float64 {
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

func embedQuery(ctx context.Context, text string) ([]float64, bool) {
	if text == "" {
		return nil, false
	}
	vec, err := ollamaEmbed(ctx, text)
	return vec, err == nil && len(vec) > 0
}

func ensureVector(ctx context.Context, id, content string) ([]float64, error) {
	if globalEmbedStore != nil {
		if rec, ok := globalEmbedStore.Get(id); ok && len(rec.Vector) > 0 {
			return rec.Vector, nil
		}
	}
	vec, err := ollamaEmbed(ctx, content)
	if err != nil {
		return nil, err
	}
	if globalEmbedStore != nil {
		_ = globalEmbedStore.Set(id, embedModel, vec)
	}
	return vec, nil
}

func ollamaEmbed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]string{"model": embedModel, "prompt": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, embedEndpoint+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := embedHTTP.Do(req)
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

// QueryPreview runs retrieval for API debug (no prompt write).
func QueryPreview(ctx context.Context, pctx PromptContext, agentID string, scope Scope) []Entry {
	g, a, c, _ := SelectForPrompt(ctx, pctx, agentID)
	switch scope {
	case ScopeGlobal:
		return g
	case ScopeCollaboration:
		return c
	case ScopeAgent:
		return a
	default:
		var all []Entry
		all = append(all, g...)
		all = append(all, a...)
		all = append(all, c...)
		return all
	}
}
