package confluence

import (
	"time"
)

// ConfluenceIndex represents the indexed state of a Confluence space
type ConfluenceIndex struct {
	SpaceKey     string               `json:"space_key"`
	SpaceName    string               `json:"space_name"`
	LastIndexed  time.Time            `json:"last_indexed"`
	PageCount    int                  `json:"page_count"`
	Pages        map[string]*Page     `json:"pages"`         // pageID -> Page
	Hierarchy    map[string][]string  `json:"hierarchy"`     // parentID -> childIDs
	Labels       map[string][]string  `json:"labels"`        // label -> pageIDs
	LastModified map[string]time.Time `json:"last_modified"` // pageID -> timestamp
	TotalSize    int64                `json:"total_size"`    // Total content size in bytes
	Description  string               `json:"description"`   // Space description
}

// Page represents an indexed Confluence page
type Page struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`     // Processed text content
	RawContent  string        `json:"raw_content"` // Original storage format
	Labels      []string      `json:"labels"`
	Comments    []PageComment `json:"comments"`
	Author      string        `json:"author"`
	LastUpdated time.Time     `json:"last_updated"`
	ParentID    string        `json:"parent_id"` // ID of parent page
	Version     int           `json:"version"`   // Page version number
	URL         string        `json:"url"`       // Full URL to page
}

// PageComment represents a comment on a page
type PageComment struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// NewConfluenceIndex creates a new empty index for a space
func NewConfluenceIndex(spaceKey, spaceName string) *ConfluenceIndex {
	return &ConfluenceIndex{
		SpaceKey:     spaceKey,
		SpaceName:    spaceName,
		LastIndexed:  time.Now(),
		PageCount:    0,
		Pages:        make(map[string]*Page),
		Hierarchy:    make(map[string][]string),
		Labels:       make(map[string][]string),
		LastModified: make(map[string]time.Time),
		TotalSize:    0,
	}
}

// AddPage adds a page to the index
func (ci *ConfluenceIndex) AddPage(page *Page) {
	ci.Pages[page.ID] = page
	ci.LastModified[page.ID] = page.LastUpdated
	ci.PageCount = len(ci.Pages)
	ci.TotalSize += int64(len(page.Content))

	// Update hierarchy
	if page.ParentID != "" {
		ci.Hierarchy[page.ParentID] = append(ci.Hierarchy[page.ParentID], page.ID)
	}

	// Update labels index
	for _, label := range page.Labels {
		ci.Labels[label] = append(ci.Labels[label], page.ID)
	}
}

// GetPage retrieves a page by ID
func (ci *ConfluenceIndex) GetPage(pageID string) (*Page, bool) {
	page, exists := ci.Pages[pageID]
	return page, exists
}

// GetPagesByLabel retrieves all pages with a specific label
func (ci *ConfluenceIndex) GetPagesByLabel(label string) []*Page {
	pageIDs, exists := ci.Labels[label]
	if !exists {
		return nil
	}

	var pages []*Page
	for _, id := range pageIDs {
		if page, exists := ci.Pages[id]; exists {
			pages = append(pages, page)
		}
	}
	return pages
}

// GetChildPages retrieves all child pages of a parent page
func (ci *ConfluenceIndex) GetChildPages(parentID string) []*Page {
	childIDs, exists := ci.Hierarchy[parentID]
	if !exists {
		return nil
	}

	var pages []*Page
	for _, id := range childIDs {
		if page, exists := ci.Pages[id]; exists {
			pages = append(pages, page)
		}
	}
	return pages
}

// GetAllPages retrieves all pages in the index
func (ci *ConfluenceIndex) GetAllPages() []*Page {
	pages := make([]*Page, 0, len(ci.Pages))
	for _, page := range ci.Pages {
		pages = append(pages, page)
	}
	return pages
}

// GetRecentlyUpdated retrieves pages updated after a given time
func (ci *ConfluenceIndex) GetRecentlyUpdated(since time.Time) []*Page {
	var pages []*Page
	for _, page := range ci.Pages {
		if page.LastUpdated.After(since) {
			pages = append(pages, page)
		}
	}
	return pages
}

// IsStale checks if the index needs to be refreshed
func (ci *ConfluenceIndex) IsStale(maxAge time.Duration) bool {
	return time.Since(ci.LastIndexed) > maxAge
}

// GetStats returns statistics about the index
func (ci *ConfluenceIndex) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"space_key":        ci.SpaceKey,
		"space_name":       ci.SpaceName,
		"page_count":       ci.PageCount,
		"label_count":      len(ci.Labels),
		"last_indexed":     ci.LastIndexed,
		"total_size_bytes": ci.TotalSize,
		"total_size_mb":    float64(ci.TotalSize) / (1024 * 1024),
	}
}
