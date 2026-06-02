package learning

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/embed"
)

type embedRecord = embed.Record

type embedFile struct {
	Entries map[string]embedRecord `json:"entries"`
}

// EmbedStore caches vectors in learning-embeddings.json.
type EmbedStore struct {
	inner *embed.Store
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
	inner, err := embed.NewStore(path)
	if err != nil {
		return nil, err
	}
	return &EmbedStore{inner: inner}, nil
}

func (es *EmbedStore) Get(id string) (embedRecord, bool) {
	return es.inner.Get(id)
}

func (es *EmbedStore) Set(id, model string, vec []float64) error {
	return es.inner.Set(id, model, vec)
}

func (es *EmbedStore) Delete(id string) {
	es.inner.Delete(id)
}

func (es *EmbedStore) Count() int {
	return es.inner.Count()
}

var (
	globalEmbedStore *EmbedStore
	embedClient      *embed.Client
	collabResolver   func(channel string) string
	onEntryChanged   func(entry Entry)
)

func SetEmbedStore(es *EmbedStore) { globalEmbedStore = es }

func SetEmbedConfig(endpoint, model string) {
	embedClient = embed.NewClient(endpoint, model)
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
		score := embed.KeywordScore(query, e.Content)
		if embedOK && globalEmbedStore != nil {
			if rec, ok := globalEmbedStore.Get(e.ID); ok && len(rec.Vector) > 0 {
				score = embed.CosineSimilarity(queryVec, rec.Vector)
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

func embedQuery(ctx context.Context, text string) ([]float64, bool) {
	if text == "" || embedClient == nil {
		return nil, false
	}
	vec, err := embedClient.Embed(ctx, text, false)
	return vec, err == nil && len(vec) > 0
}

func ensureVector(ctx context.Context, id, content string) ([]float64, error) {
	if globalEmbedStore != nil {
		if rec, ok := globalEmbedStore.Get(id); ok && len(rec.Vector) > 0 {
			return rec.Vector, nil
		}
	}
	if embedClient == nil {
		return nil, nil
	}
	vec, err := embedClient.Embed(ctx, content, true)
	if err != nil {
		return nil, err
	}
	if globalEmbedStore != nil {
		model := embed.DefaultModel
		if embedClient != nil {
			model = embedClient.Model
		}
		_ = globalEmbedStore.Set(id, model, vec)
	}
	return vec, nil
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
