package confluence

import (
	"sort"
	"strings"
	"time"
)

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Page  *Page
	Score float64
	// Snippet is a relevant excerpt from the content
	Snippet string
}

// Searcher provides search functionality for Confluence indexes
type Searcher struct {
	index *ConfluenceIndex
}

// NewSearcher creates a new searcher for an index
func NewSearcher(index *ConfluenceIndex) *Searcher {
	return &Searcher{
		index: index,
	}
}

// Search performs a full-text search across pages
func (s *Searcher) Search(query string, limit int) []SearchResult {
	if query == "" {
		return []SearchResult{}
	}

	query = strings.ToLower(strings.TrimSpace(query))
	queryTerms := strings.Fields(query)

	var results []SearchResult

	// Search through all pages
	for _, page := range s.index.Pages {
		score := s.calculateRelevance(page, queryTerms)
		if score > 0 {
			snippet := s.extractSnippet(page.Content, queryTerms, 200)
			results = append(results, SearchResult{
				Page:    page,
				Score:   score,
				Snippet: snippet,
			})
		}
	}

	// Sort by relevance score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// SearchByTitle searches for pages by title
func (s *Searcher) SearchByTitle(query string, limit int) []SearchResult {
	if query == "" {
		return []SearchResult{}
	}

	query = strings.ToLower(strings.TrimSpace(query))
	var results []SearchResult

	for _, page := range s.index.Pages {
		title := strings.ToLower(page.Title)
		if strings.Contains(title, query) {
			// Calculate title match score
			score := 100.0
			if title == query {
				score = 150.0 // Exact match
			}

			results = append(results, SearchResult{
				Page:    page,
				Score:   score,
				Snippet: page.Title,
			})
		}
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// SearchByLabel searches for pages by label
func (s *Searcher) SearchByLabel(label string) []SearchResult {
	pages := s.index.GetPagesByLabel(label)

	var results []SearchResult
	for _, page := range pages {
		results = append(results, SearchResult{
			Page:    page,
			Score:   50.0, // Fixed score for label matches
			Snippet: "Labeled: " + label,
		})
	}

	return results
}

// SearchByDateRange searches for pages updated within a date range
func (s *Searcher) SearchByDateRange(start, end time.Time) []SearchResult {
	var results []SearchResult

	for _, page := range s.index.Pages {
		if page.LastUpdated.After(start) && page.LastUpdated.Before(end) {
			results = append(results, SearchResult{
				Page:    page,
				Score:   25.0, // Fixed score for date matches
				Snippet: "Updated: " + page.LastUpdated.Format("2006-01-02"),
			})
		}
	}

	// Sort by date (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Page.LastUpdated.After(results[j].Page.LastUpdated)
	})

	return results
}

// SearchInComments searches for text in page comments
func (s *Searcher) SearchInComments(query string, limit int) []SearchResult {
	if query == "" {
		return []SearchResult{}
	}

	query = strings.ToLower(strings.TrimSpace(query))
	queryTerms := strings.Fields(query)

	var results []SearchResult

	for _, page := range s.index.Pages {
		for _, comment := range page.Comments {
			content := strings.ToLower(comment.Content)
			score := 0.0

			for _, term := range queryTerms {
				if strings.Contains(content, term) {
					score += 10.0 // Lower score for comment matches
				}
			}

			if score > 0 {
				snippet := s.extractSnippet(comment.Content, queryTerms, 150)
				results = append(results, SearchResult{
					Page:    page,
					Score:   score,
					Snippet: "Comment: " + snippet,
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// GetMostRecentPages returns the most recently updated pages
func (s *Searcher) GetMostRecentPages(limit int) []SearchResult {
	var results []SearchResult

	for _, page := range s.index.Pages {
		results = append(results, SearchResult{
			Page:    page,
			Score:   float64(page.LastUpdated.Unix()),
			Snippet: "Updated: " + page.LastUpdated.Format("2006-01-02 15:04"),
		})
	}

	// Sort by last updated (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Page.LastUpdated.After(results[j].Page.LastUpdated)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// calculateRelevance calculates a relevance score for a page given query terms
func (s *Searcher) calculateRelevance(page *Page, queryTerms []string) float64 {
	score := 0.0

	title := strings.ToLower(page.Title)
	content := strings.ToLower(page.Content)

	for _, term := range queryTerms {
		// Title matches are worth more
		titleCount := float64(strings.Count(title, term))
		score += titleCount * 10.0

		// Content matches
		contentCount := float64(strings.Count(content, term))
		score += contentCount * 1.0

		// Label matches
		for _, label := range page.Labels {
			if strings.Contains(strings.ToLower(label), term) {
				score += 5.0
			}
		}
	}

	// Boost score for recent updates (within 30 days)
	daysSinceUpdate := time.Since(page.LastUpdated).Hours() / 24
	if daysSinceUpdate < 30 {
		score *= 1.2
	}

	return score
}

// extractSnippet extracts a relevant snippet from content around query terms
func (s *Searcher) extractSnippet(content string, queryTerms []string, maxLength int) string {
	content = strings.TrimSpace(content)

	if len(content) <= maxLength {
		return content
	}

	// Find first occurrence of any query term
	lowerContent := strings.ToLower(content)
	firstIndex := -1

	for _, term := range queryTerms {
		index := strings.Index(lowerContent, strings.ToLower(term))
		if index != -1 && (firstIndex == -1 || index < firstIndex) {
			firstIndex = index
		}
	}

	if firstIndex == -1 {
		// No terms found, return beginning
		if len(content) > maxLength {
			return content[:maxLength] + "..."
		}
		return content
	}

	// Extract snippet around the found term
	start := firstIndex - maxLength/2
	if start < 0 {
		start = 0
	}

	end := start + maxLength
	if end > len(content) {
		end = len(content)
		start = end - maxLength
		if start < 0 {
			start = 0
		}
	}

	snippet := content[start:end]

	// Add ellipsis if needed
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return strings.TrimSpace(snippet)
}

// FindRelatedPages finds pages related to a given page (by labels and hierarchy)
func (s *Searcher) FindRelatedPages(pageID string, limit int) []SearchResult {
	page, exists := s.index.GetPage(pageID)
	if !exists {
		return []SearchResult{}
	}

	scoreMap := make(map[string]float64)

	// Find pages with same labels
	for _, label := range page.Labels {
		relatedPages := s.index.GetPagesByLabel(label)
		for _, related := range relatedPages {
			if related.ID != pageID {
				scoreMap[related.ID] += 10.0
			}
		}
	}

	// Find sibling pages (same parent)
	if page.ParentID != "" {
		siblings := s.index.GetChildPages(page.ParentID)
		for _, sibling := range siblings {
			if sibling.ID != pageID {
				scoreMap[sibling.ID] += 15.0
			}
		}
	}

	// Find child pages
	children := s.index.GetChildPages(pageID)
	for _, child := range children {
		scoreMap[child.ID] += 20.0
	}

	// Build results
	var results []SearchResult
	for id, score := range scoreMap {
		if relatedPage, exists := s.index.GetPage(id); exists {
			results = append(results, SearchResult{
				Page:    relatedPage,
				Score:   score,
				Snippet: "Related to: " + page.Title,
			})
		}
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}
