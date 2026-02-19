package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	// MaxPageSize is the maximum number of results per API call
	MaxPageSize = 100
	// RateLimit is the delay between API calls to respect rate limits
	RateLimit = 200 * time.Millisecond
	// MaxRetries is the number of times to retry failed requests
	MaxRetries = 3
)

// Client represents a Confluence Cloud API client
type Client struct {
	BaseURL    string
	Email      string
	APIToken   string
	HTTPClient *http.Client
	lastCall   time.Time
}

// NewClient creates a new Confluence API client
func NewClient() (*Client, error) {
	domain := os.Getenv("CONFLUENCE_DOMAIN")
	email := os.Getenv("CONFLUENCE_EMAIL")
	apiToken := os.Getenv("CONFLUENCE_API_TOKEN")

	if domain == "" || email == "" || apiToken == "" {
		return nil, fmt.Errorf("missing Confluence credentials: CONFLUENCE_DOMAIN, CONFLUENCE_EMAIL, and CONFLUENCE_API_TOKEN must be set")
	}

	// Build base URL
	baseURL := fmt.Sprintf("https://%s/wiki/rest/api", domain)

	return &Client{
		BaseURL:  baseURL,
		Email:    email,
		APIToken: apiToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SetCredentials updates the client credentials
func (c *Client) SetCredentials(credentials map[string]string) {
	if domain, ok := credentials["domain"]; ok && domain != "" {
		c.BaseURL = fmt.Sprintf("https://%s/wiki/rest/api", domain)
	}
	if email, ok := credentials["email"]; ok && email != "" {
		c.Email = email
	}
	if apiToken, ok := credentials["api_token"]; ok && apiToken != "" {
		c.APIToken = apiToken
	}
}

// rateLimit ensures we don't exceed API rate limits
func (c *Client) rateLimit() {
	since := time.Since(c.lastCall)
	if since < RateLimit {
		time.Sleep(RateLimit - since)
	}
	c.lastCall = time.Now()
}

// doRequest performs an HTTP request with authentication and rate limiting
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	c.rateLimit()

	reqURL := c.BaseURL + path
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic authentication
	req.SetBasicAuth(c.Email, c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request with retries
	var resp *http.Response
	for i := 0; i < MaxRetries; i++ {
		resp, err = c.HTTPClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if i < MaxRetries-1 {
			// Exponential backoff
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after retries: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// SpaceResponse represents a Confluence space
type SpaceResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description struct {
		Plain struct {
			Value string `json:"value"`
		} `json:"plain"`
	} `json:"description"`
}

// GetSpace retrieves information about a space
func (c *Client) GetSpace(spaceKey string) (*SpaceResponse, error) {
	path := fmt.Sprintf("/space/%s?expand=description.plain", url.PathEscape(spaceKey))

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var space SpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&space); err != nil {
		return nil, fmt.Errorf("failed to decode space: %w", err)
	}

	return &space, nil
}

// PageResponse represents a Confluence page
type PageResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Version struct {
		Number int       `json:"number"`
		When   time.Time `json:"when"`
		By     User      `json:"by"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
		View struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"view"`
	} `json:"body"`
	Metadata struct {
		Labels struct {
			Results []Label `json:"results"`
		} `json:"labels"`
	} `json:"metadata"`
	Ancestors []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"ancestors"`
}

// User represents a Confluence user
type User struct {
	Type        string `json:"type"`
	Username    string `json:"username"`
	UserKey     string `json:"userKey"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// Label represents a Confluence label
type Label struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
}

// Comment represents a Confluence comment
type Comment struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Version struct {
		Number int       `json:"number"`
		When   time.Time `json:"when"`
		By     User      `json:"by"`
	} `json:"version"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Results []json.RawMessage `json:"results"`
	Start   int               `json:"start"`
	Limit   int               `json:"limit"`
	Size    int               `json:"size"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}

// GetPages retrieves all pages in a space
func (c *Client) GetPages(spaceKey string) ([]PageResponse, error) {
	var allPages []PageResponse
	start := 0

	for {
		path := fmt.Sprintf("/content?spaceKey=%s&type=page&limit=%d&start=%d&expand=body.storage,version,metadata.labels,ancestors",
			url.QueryEscape(spaceKey), MaxPageSize, start)

		resp, err := c.doRequest("GET", path, nil)
		if err != nil {
			return nil, err
		}

		var paginated PaginatedResponse
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, &paginated); err != nil {
			return nil, fmt.Errorf("failed to decode pages: %w", err)
		}

		// Parse individual pages
		for _, rawPage := range paginated.Results {
			var page PageResponse
			if err := json.Unmarshal(rawPage, &page); err != nil {
				continue // Skip malformed pages
			}
			allPages = append(allPages, page)
		}

		// Check if there are more pages
		if len(paginated.Results) < paginated.Limit {
			break
		}
		start += paginated.Limit
	}

	return allPages, nil
}

// GetPageContent retrieves full content for a specific page
func (c *Client) GetPageContent(pageID string) (*PageResponse, error) {
	path := fmt.Sprintf("/content/%s?expand=body.storage,body.view,version,metadata.labels,ancestors",
		url.PathEscape(pageID))

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var page PageResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode page: %w", err)
	}

	return &page, nil
}

// GetComments retrieves comments for a page
func (c *Client) GetComments(pageID string) ([]Comment, error) {
	var allComments []Comment
	start := 0

	for {
		path := fmt.Sprintf("/content/%s/child/comment?limit=%d&start=%d&expand=body.storage,version",
			url.PathEscape(pageID), MaxPageSize, start)

		resp, err := c.doRequest("GET", path, nil)
		if err != nil {
			return nil, err
		}

		var paginated PaginatedResponse
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, &paginated); err != nil {
			return nil, fmt.Errorf("failed to decode comments: %w", err)
		}

		// Parse individual comments
		for _, rawComment := range paginated.Results {
			var comment Comment
			if err := json.Unmarshal(rawComment, &comment); err != nil {
				continue // Skip malformed comments
			}
			allComments = append(allComments, comment)
		}

		// Check if there are more comments
		if len(paginated.Results) < paginated.Limit {
			break
		}
		start += paginated.Limit
	}

	return allComments, nil
}

// CreateComment creates a comment on a page (example write operation)
func (c *Client) CreateComment(pageID, content string) (*Comment, error) {
	commentData := map[string]interface{}{
		"type": "comment",
		"container": map[string]string{
			"id":   pageID,
			"type": "page",
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          content,
				"representation": "storage",
			},
		},
	}

	jsonData, err := json.Marshal(commentData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	resp, err := c.doRequest("POST", "/content", bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var comment Comment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode comment: %w", err)
	}

	return &comment, nil
}
