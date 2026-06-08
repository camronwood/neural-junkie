package memory

import (
	"context"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/embed"
)

// Search finds relevant chunks for a query within channel/collab scope.
func Search(ctx context.Context, pctx PromptContext, limit int) ([]SearchResult, error) {
	if !memoryEnabled() {
		return nil, nil
	}
	query := strings.TrimSpace(pctx.Query)
	channel := strings.TrimSpace(pctx.Channel)
	collabID := strings.TrimSpace(pctx.CollaborationID)
	if collabID == "" && channel != "" {
		collabID = ResolveCollabID(channel)
	}
	if query == "" || (channel == "" && collabID == "") {
		return nil, nil
	}
	if limit <= 0 {
		limit = DefaultTopK
	}

	candidates, err := globalStore.ListCandidates(channel, collabID, DefaultSearchPrefilter)
	if err != nil {
		return nil, err
	}
	exclude := excludeSet(pctx.ExcludeMessageIDs)
	filtered := make([]Chunk, 0, len(candidates))
	for _, ch := range candidates {
		if ch.SourceType == SourceMessage && exclude[ch.SourceID] {
			continue
		}
		filtered = append(filtered, ch)
	}
	if len(filtered) == 0 {
		return nil, nil
	}

	queryVec, embedOK := embedQuery(ctx, query)
	scored := make([]embed.ScoredItem[SearchResult], 0, len(filtered))
	for _, ch := range filtered {
		score := embed.KeywordScore(query, ch.Content+" "+ch.RelPath+" "+ch.SenderName)
		if embedOK && len(queryVec) > 0 && len(ch.Vector) > 0 {
			score = embed.CosineSimilarity(queryVec, ch.Vector)
		}
		if score <= 0 {
			continue
		}
		scored = append(scored, embed.ScoredItem[SearchResult]{
			Item:  SearchResult{Chunk: ch, Score: score},
			Score: score,
		})
	}
	if len(scored) == 0 {
		return nil, nil
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > limit {
		scored = scored[:limit]
	}
	out := make([]SearchResult, len(scored))
	for i, s := range scored {
		out[i] = s.Item
	}
	return out, nil
}

func excludeSet(ids []string) map[string]bool {
	out := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = true
		}
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

// QueryPreview runs retrieval for API debug.
func QueryPreview(ctx context.Context, pctx PromptContext, limit int) ([]SearchResult, error) {
	return Search(ctx, pctx, limit)
}
