package confluence

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// MaxContentSize is the maximum size for a single page content (10MB)
	MaxContentSize = 10 * 1024 * 1024
	// MaxTotalSize is the maximum total size for all pages (500MB)
	MaxTotalSize = 500 * 1024 * 1024
)

// Analyzer analyzes and indexes Confluence spaces
type Analyzer struct {
	client           *Client
	progressCallback func(progress int, message string)
}

// NewAnalyzer creates a new Confluence analyzer
func NewAnalyzer(client *Client, progressCallback func(progress int, message string)) *Analyzer {
	return &Analyzer{
		client:           client,
		progressCallback: progressCallback,
	}
}

// AnalyzeSpace analyzes a Confluence space and creates an index
func (a *Analyzer) AnalyzeSpace(ctx context.Context, spaceKey string) (*ConfluenceIndex, error) {
	a.updateProgress(0, "Starting space analysis...")

	// Step 1: Get space information
	a.updateProgress(5, "Fetching space information...")
	space, err := a.client.GetSpace(spaceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get space: %w", err)
	}

	// Create index
	index := NewConfluenceIndex(space.Key, space.Name)
	index.Description = space.Description.Plain.Value

	// Step 2: Get all pages in the space
	a.updateProgress(10, "Fetching pages from space...")
	pages, err := a.client.GetPages(spaceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}

	if len(pages) == 0 {
		a.updateProgress(100, "Space is empty")
		return index, nil
	}

	a.updateProgress(20, fmt.Sprintf("Found %d pages, processing content...", len(pages)))

	// Step 3: Process each page
	totalPages := len(pages)
	var totalSize int64

	for i, pageResp := range pages {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate progress (20% to 80% for page processing)
		progress := 20 + (i * 60 / totalPages)
		a.updateProgress(progress, fmt.Sprintf("Processing page %d/%d: %s", i+1, totalPages, pageResp.Title))

		// Process page content
		page := &Page{
			ID:          pageResp.ID,
			Title:       pageResp.Title,
			RawContent:  pageResp.Body.Storage.Value,
			Content:     a.extractTextContent(pageResp.Body.Storage.Value),
			Author:      pageResp.Version.By.DisplayName,
			LastUpdated: pageResp.Version.When,
			Version:     pageResp.Version.Number,
			URL:         fmt.Sprintf("https://%s/wiki/spaces/%s/pages/%s", strings.TrimPrefix(a.client.BaseURL, "https://"), spaceKey, pageResp.ID),
		}

		// Extract labels
		for _, label := range pageResp.Metadata.Labels.Results {
			page.Labels = append(page.Labels, label.Name)
		}

		// Determine parent
		if len(pageResp.Ancestors) > 0 {
			page.ParentID = pageResp.Ancestors[len(pageResp.Ancestors)-1].ID
		}

		// Check content size
		contentSize := int64(len(page.Content))
		if contentSize > MaxContentSize {
			page.Content = page.Content[:MaxContentSize]
			contentSize = MaxContentSize
		}

		totalSize += contentSize
		if totalSize > MaxTotalSize {
			return nil, fmt.Errorf("space too large: total content exceeds %d bytes", MaxTotalSize)
		}

		// Add page to index (without comments for now)
		index.AddPage(page)
	}

	// Step 4: Fetch comments for pages
	a.updateProgress(80, "Fetching comments...")
	commentCount := 0
	for i, page := range index.GetAllPages() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Progress 80% to 95%
		progress := 80 + (i * 15 / len(index.GetAllPages()))
		a.updateProgress(progress, fmt.Sprintf("Fetching comments for page %d/%d", i+1, len(index.GetAllPages())))

		comments, err := a.client.GetComments(page.ID)
		if err != nil {
			// Non-fatal: log and continue
			continue
		}

		for _, comment := range comments {
			pageComment := PageComment{
				ID:        comment.ID,
				Content:   a.extractTextContent(comment.Body),
				Author:    comment.Version.By.DisplayName,
				CreatedAt: comment.Version.When,
			}
			page.Comments = append(page.Comments, pageComment)
			commentCount++
		}
	}

	// Step 5: Finalize index
	a.updateProgress(95, "Finalizing index...")
	index.LastIndexed = time.Now()

	a.updateProgress(100, fmt.Sprintf("Indexing complete: %d pages, %d comments", index.PageCount, commentCount))

	return index, nil
}

// extractTextContent extracts plain text from Confluence storage format (HTML)
func (a *Analyzer) extractTextContent(storageContent string) string {
	// Remove HTML tags
	content := a.stripHTMLTags(storageContent)

	// Decode HTML entities
	content = a.decodeHTMLEntities(content)

	// Normalize whitespace
	content = a.normalizeWhitespace(content)

	return strings.TrimSpace(content)
}

// stripHTMLTags removes HTML tags from content
func (a *Analyzer) stripHTMLTags(html string) string {
	// Remove script and style tags with their content
	scriptRegex := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRegex.ReplaceAllString(html, "")

	styleRegex := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRegex.ReplaceAllString(html, "")

	// Remove all HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text := tagRegex.ReplaceAllString(html, " ")

	return text
}

// decodeHTMLEntities decodes common HTML entities
func (a *Analyzer) decodeHTMLEntities(text string) string {
	replacements := map[string]string{
		"&nbsp;":   " ",
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&#39;":    "'",
		"&apos;":   "'",
		"&ndash;":  "-",
		"&mdash;":  "-",
		"&hellip;": "...",
	}

	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	return text
}

// normalizeWhitespace normalizes whitespace in text
func (a *Analyzer) normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	// Replace multiple newlines with double newline
	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	return text
}

// updateProgress calls the progress callback if set
func (a *Analyzer) updateProgress(progress int, message string) {
	if a.progressCallback != nil {
		a.progressCallback(progress, message)
	}
}

// CheckStaleness checks if an index needs to be refreshed
func (a *Analyzer) CheckStaleness(index *ConfluenceIndex) (bool, []string, error) {
	// Fetch current pages from space
	pages, err := a.client.GetPages(index.SpaceKey)
	if err != nil {
		return false, nil, fmt.Errorf("failed to fetch pages: %w", err)
	}

	// Check for new or updated pages
	var stalePagesIds []string
	for _, pageResp := range pages {
		existingPage, exists := index.GetPage(pageResp.ID)
		if !exists {
			// New page
			stalePagesIds = append(stalePagesIds, pageResp.ID)
			continue
		}

		// Check if updated
		if pageResp.Version.When.After(existingPage.LastUpdated) {
			stalePagesIds = append(stalePagesIds, pageResp.ID)
		}
	}

	// Check for deleted pages
	for pageID := range index.Pages {
		found := false
		for _, pageResp := range pages {
			if pageResp.ID == pageID {
				found = true
				break
			}
		}
		if !found {
			stalePagesIds = append(stalePagesIds, pageID)
		}
	}

	isStale := len(stalePagesIds) > 0
	return isStale, stalePagesIds, nil
}

// IncrementalUpdate updates only changed pages in an index
func (a *Analyzer) IncrementalUpdate(ctx context.Context, index *ConfluenceIndex, pageIDs []string) error {
	totalPages := len(pageIDs)

	for i, pageID := range pageIDs {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		progress := (i * 100 / totalPages)
		a.updateProgress(progress, fmt.Sprintf("Updating page %d/%d", i+1, totalPages))

		// Fetch updated page content
		pageResp, err := a.client.GetPageContent(pageID)
		if err != nil {
			// Page might have been deleted
			if _, exists := index.Pages[pageID]; exists {
				delete(index.Pages, pageID)
				delete(index.LastModified, pageID)
			}
			continue
		}

		// Process page
		page := &Page{
			ID:          pageResp.ID,
			Title:       pageResp.Title,
			RawContent:  pageResp.Body.Storage.Value,
			Content:     a.extractTextContent(pageResp.Body.Storage.Value),
			Author:      pageResp.Version.By.DisplayName,
			LastUpdated: pageResp.Version.When,
			Version:     pageResp.Version.Number,
			URL:         fmt.Sprintf("https://%s/wiki/spaces/%s/pages/%s", strings.TrimPrefix(a.client.BaseURL, "https://"), index.SpaceKey, pageResp.ID),
		}

		// Extract labels
		for _, label := range pageResp.Metadata.Labels.Results {
			page.Labels = append(page.Labels, label.Name)
		}

		// Determine parent
		if len(pageResp.Ancestors) > 0 {
			page.ParentID = pageResp.Ancestors[len(pageResp.Ancestors)-1].ID
		}

		// Fetch comments
		comments, err := a.client.GetComments(page.ID)
		if err == nil {
			for _, comment := range comments {
				pageComment := PageComment{
					ID:        comment.ID,
					Content:   a.extractTextContent(comment.Body),
					Author:    comment.Version.By.DisplayName,
					CreatedAt: comment.Version.When,
				}
				page.Comments = append(page.Comments, pageComment)
			}
		}

		// Update index
		index.AddPage(page)
	}

	index.LastIndexed = time.Now()
	a.updateProgress(100, fmt.Sprintf("Updated %d pages", totalPages))

	return nil
}
