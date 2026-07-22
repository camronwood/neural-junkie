package memory

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

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
	lexicalCandidates, ftsScores := globalStore.LexicalCandidates(query, channel, collabID, DefaultSearchPrefilter)
	byID := make(map[string]Chunk, len(candidates)+len(lexicalCandidates))
	for _, ch := range candidates {
		byID[ch.ID] = ch
	}
	for _, ch := range lexicalCandidates {
		byID[ch.ID] = ch
	}
	candidates = candidates[:0]
	for _, ch := range byID {
		candidates = append(candidates, ch)
	}
	exclude := excludeSet(pctx.ExcludeMessageIDs)
	for id := range excludeSet(pctx.SupersededMessageIDs) {
		exclude[id] = true
	}
	sourceAllow := sourceTypeAllowSet(pctx.SourceTypes)
	filtered := make([]Chunk, 0, len(candidates))
	for _, ch := range candidates {
		if ch.SourceType == SourceMessage && exclude[ch.SourceID] {
			continue
		}
		if sourceAllow != nil && !sourceAllow[ch.SourceType] {
			continue
		}
		// Corrections are authoritative only inside the goal they amended.
		if ch.IsCorrection && pctx.GoalID != "" && ch.GoalID != "" && ch.GoalID != pctx.GoalID {
			continue
		}
		filtered = append(filtered, ch)
	}
	if len(filtered) == 0 {
		return nil, nil
	}

	queryVec, embedOK := embedQuery(ctx, query)
	scored := make([]embed.ScoredItem[SearchResult], 0, len(filtered))
	now := time.Now()
	for _, ch := range filtered {
		searchable := ch.Content + " " + ch.RelPath + " " + ch.SenderName
		keyword := embed.KeywordScore(query, searchable)
		lexical := math.Max(keyword, lexicalCoverage(query, searchable))
		if fts := ftsScores[ch.ID]; fts > 0 {
			// BM25 rank refines lexical ordering without allowing a single
			// incidental OR-term match to overwhelm query-term coverage.
			lexical *= 0.7 + 0.3*fts
		}
		vectorScore := 0.0
		if embedOK && len(queryVec) > 0 && len(ch.Vector) > 0 {
			vectorScore = math.Max(0, embed.CosineSimilarity(queryVec, ch.Vector))
		}
		recency := recencyScore(now, ch.CreatedAt)
		score := 0.5*lexical + 0.4*vectorScore + 0.1*recency
		if pctx.ThreadID != "" && ch.ThreadID == pctx.ThreadID {
			score += 0.12
		}
		if pctx.GoalID != "" && ch.GoalID == pctx.GoalID {
			score += 0.08
		}
		if ch.IsCorrection {
			score += 0.04
		}
		if score < DefaultRelevanceFloor {
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
		iThread := pctx.ThreadID != "" && scored[i].Item.Chunk.ThreadID == pctx.ThreadID
		jThread := pctx.ThreadID != "" && scored[j].Item.Chunk.ThreadID == pctx.ThreadID
		if iThread != jThread {
			return iThread
		}
		if scored[i].Score == scored[j].Score {
			if scored[i].Item.Chunk.CreatedAt.Equal(scored[j].Item.Chunk.CreatedAt) {
				return scored[i].Item.Chunk.ID < scored[j].Item.Chunk.ID
			}
			return scored[i].Item.Chunk.CreatedAt.After(scored[j].Item.Chunk.CreatedAt)
		}
		return scored[i].Score > scored[j].Score
	})
	out := make([]SearchResult, 0, limit)
	seenSources := make(map[string]bool)
	for _, candidate := range scored {
		ch := candidate.Item.Chunk
		sourceKey := string(ch.SourceType) + "\x00" + ch.SourceID
		if seenSources[sourceKey] || tooSimilarToSelected(ch.Content, out) {
			continue
		}
		seenSources[sourceKey] = true
		out = append(out, candidate.Item)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func recencyScore(now, created time.Time) float64 {
	if created.IsZero() || created.After(now) {
		return 1
	}
	const halfLife = 30 * 24 * time.Hour
	return math.Exp(-math.Ln2 * float64(now.Sub(created)) / float64(halfLife))
}

func tooSimilarToSelected(content string, selected []SearchResult) bool {
	terms := termSet(content)
	if len(terms) == 0 {
		return false
	}
	for _, existing := range selected {
		other := termSet(existing.Chunk.Content)
		common := 0
		for term := range terms {
			if other[term] {
				common++
			}
		}
		union := len(terms) + len(other) - common
		if union > 0 && float64(common)/float64(union) >= 0.8 {
			return true
		}
	}
	return false
}

func termSet(text string) map[string]bool {
	out := make(map[string]bool)
	for _, term := range strings.Fields(strings.ToLower(text)) {
		term = strings.Trim(term, " \t\r\n.,:;!?()[]{}<>\"'`")
		if len(term) > 2 {
			out[term] = true
		}
	}
	return out
}

func lexicalCoverage(query, content string) float64 {
	queryTerms := termSet(query)
	if len(queryTerms) == 0 {
		return 0
	}
	contentTerms := termSet(content)
	hits := 0
	for term := range queryTerms {
		if contentTerms[term] {
			hits++
		}
	}
	return float64(hits) / float64(len(queryTerms))
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

func sourceTypeAllowSet(types []SourceType) map[SourceType]bool {
	if len(types) == 0 {
		return nil
	}
	out := make(map[SourceType]bool, len(types))
	for _, st := range types {
		out[st] = true
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
