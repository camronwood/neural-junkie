package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ClaudeProvider implements AI responses using Claude API or AI Hub
type ClaudeProvider struct {
	APIKey     string
	Model      string
	BaseURL    string
	UseAIHub   bool
	httpClient *http.Client
}

// NewClaudeProvider creates a new Claude AI provider
func NewClaudeProvider() (*ClaudeProvider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Check if we should use AI Hub proxy
	useAIHub := os.Getenv("USE_AI_HUB") == "true"
	baseURL := "https://api.anthropic.com/v1"
	model := "claude-3-5-sonnet-20241022"

	if useAIHub {
		// Use AI Hub proxy endpoint
		aiHubEndpoint := os.Getenv("AI_HUB_ENDPOINT")
		if aiHubEndpoint == "" {
			aiHubEndpoint = "https://aihub.dispatchit.com/v1"
		}
		baseURL = aiHubEndpoint

		// Use AI Hub model names
		aiHubModel := os.Getenv("AI_HUB_MODEL")
		if aiHubModel != "" {
			model = aiHubModel
		} else {
			model = "claude-sonnet" // Default AI Hub model
		}
	}

	return &ClaudeProvider{
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
		UseAIHub: useAIHub,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// NewClaudeProviderWithConfig creates a new Claude AI provider with custom configuration
func NewClaudeProviderWithConfig(apiKey string, useAIHub bool, aiHubEndpoint, model string) *ClaudeProvider {
	baseURL := "https://api.anthropic.com/v1"

	if useAIHub && aiHubEndpoint != "" {
		baseURL = aiHubEndpoint
	}

	if model == "" {
		if useAIHub {
			model = "claude-sonnet"
		} else {
			model = "claude-3-5-sonnet-20241022"
		}
	}

	return &ClaudeProvider{
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
		UseAIHub: useAIHub,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ClaudeMessage represents a message in the Claude API format
type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or []ClaudeContentBlock
}

// ClaudeContentBlock represents a content block in Claude API
type ClaudeContentBlock struct {
	Type   string             `json:"type"`
	Text   string             `json:"text,omitempty"`
	Source *ClaudeImageSource `json:"source,omitempty"`
}

// ClaudeImageSource represents an image source in Claude API
type ClaudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"` // base64 encoded image data
}

// ClaudeRequest represents a request to Claude API
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
}

// ClaudeResponse represents a response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
}

// GenerateResponse generates a response using Claude API or AI Hub
func (c *ClaudeProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	// Split system prompt from user message -- Claude natively supports system messages
	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	// Build messages array
	messages := []ClaudeMessage{}

	// Add conversation history
	for _, msg := range conversationHistory {
		if len(messages) >= 10 { // Limit history
			break
		}
		role := "user"
		if msg.From.Type != protocol.AgentTypeGeneral {
			role = "assistant"
		}
		messages = append(messages, ClaudeMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, ClaudeMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Use higher token limit for detailed analysis requests
	maxTokens := 1024
	if isDeepAnalysisPrompt(userMessage) {
		maxTokens = 4096
	}

	request := ClaudeRequest{
		Model:     c.Model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    systemPrompt,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.UseAIHub {
		// AI Hub authentication
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		// Direct Anthropic API authentication
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return response.Content[0].Text, nil
}

// GenerateVisionResponse generates a response using Claude API with image input
func (c *ClaudeProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	// Build messages array
	messages := []ClaudeMessage{}

	// Add conversation history
	for _, msg := range conversationHistory {
		if len(messages) >= 10 { // Limit history
			break
		}
		role := "user"
		if msg.From.Type != protocol.AgentTypeGeneral {
			role = "assistant"
		}
		messages = append(messages, ClaudeMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Create content blocks for current message with image
	contentBlocks := []ClaudeContentBlock{
		{
			Type: "text",
			Text: prompt,
		},
		{
			Type: "image",
			Source: &ClaudeImageSource{
				Type:      "base64",
				MediaType: imageType,
				Data:      string(imageData),
			},
		},
	}

	// Add current prompt with image as user message
	messages = append(messages, ClaudeMessage{
		Role:    "user",
		Content: contentBlocks,
	})

	request := ClaudeRequest{
		Model:     c.Model,
		MaxTokens: 2000, // Increased for design analysis
		Messages:  messages,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.UseAIHub {
		// AI Hub authentication
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		// Direct Anthropic API authentication
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return response.Content[0].Text, nil
}

// GetModel returns the model name
func (c *ClaudeProvider) GetModel() string {
	return c.Model
}

// generateMockResponse generates intelligent mock responses for demo purposes
func (c *ClaudeProvider) generateMockResponse(prompt string, history []protocol.Message) string {
	lower := strings.ToLower(prompt)

	// Frontend agent responses
	if strings.Contains(lower, "frontend") || strings.Contains(lower, "react") || strings.Contains(lower, "ui") {
		if strings.Contains(lower, "performance") {
			return "For frontend performance, I recommend: 1) Use React.memo() for expensive components, 2) Implement lazy loading with React.lazy() and Suspense, 3) Optimize bundle size with code splitting. Would you like me to elaborate on any of these?"
		}
		if strings.Contains(lower, "state") {
			return "For state management, consider using Context API for simple cases, Redux Toolkit for complex apps, or Zustand for a lightweight alternative. What's your use case?"
		}
		return "I can help with that! For modern frontend development, I'd suggest using TypeScript for type safety, component libraries like Shadcn UI or MUI, and Tailwind CSS for styling. What specific aspect would you like to focus on?"
	}

	// Backend agent responses
	if strings.Contains(lower, "backend") || strings.Contains(lower, "api") || strings.Contains(lower, "server") {
		if strings.Contains(lower, "slow") || strings.Contains(lower, "performance") {
			return "Let's investigate the performance issue. Common causes: 1) N+1 queries, 2) Missing database indexes, 3) Unoptimized algorithms, 4) Lack of caching. I'd recommend profiling the endpoint first. @Database Expert, can you check the query patterns?"
		}
		if strings.Contains(lower, "authentication") || strings.Contains(lower, "auth") {
			return "For authentication, I recommend implementing JWT tokens with refresh token rotation. Store access tokens in memory and refresh tokens in httpOnly cookies. @Security Expert should review the implementation for vulnerabilities."
		}
		return "I can help with the backend architecture. For a scalable API, consider: REST for simplicity, GraphQL for flexible queries, or gRPC for high-performance service-to-service communication. What are your requirements?"
	}

	// DevOps agent responses
	if strings.Contains(lower, "devops") || strings.Contains(lower, "deploy") || strings.Contains(lower, "docker") {
		if strings.Contains(lower, "ci/cd") {
			return "For CI/CD, I recommend GitHub Actions for simplicity. Pipeline should include: 1) Linting & tests, 2) Build Docker images, 3) Security scanning, 4) Deploy to staging, 5) Smoke tests, 6) Production deployment with rollback capability."
		}
		if strings.Contains(lower, "monitoring") {
			return "Set up comprehensive monitoring: Use Prometheus for metrics, Grafana for dashboards, and Loki for logs. Key metrics to track: response time, error rate, resource utilization, and business KPIs."
		}
		return "For deployment, I'd suggest containerizing with Docker, orchestrating with Kubernetes, and using Helm charts for configuration management. Need help with any specific part?"
	}

	// Database agent responses
	if strings.Contains(lower, "database") || strings.Contains(lower, "query") || strings.Contains(lower, "schema") {
		if strings.Contains(lower, "slow") || strings.Contains(lower, "n+1") {
			return "I see the N+1 query issue. Solution: Use JOIN queries or implement eager loading. For your user model, add: `SELECT users.*, posts.* FROM users LEFT JOIN posts ON users.id = posts.user_id`. I can also add a composite index on (user_id, created_at) for better performance."
		}
		if strings.Contains(lower, "migration") {
			return "For database migrations, use a tool like golang-migrate or Flyway. Best practices: 1) Make migrations reversible, 2) Test on staging first, 3) Never modify existing migrations, 4) Include data migrations separately from schema changes."
		}
		return "I can help optimize your database schema. What specific issue are you facing with queries or data modeling?"
	}

	// Security agent responses
	if strings.Contains(lower, "security") || strings.Contains(lower, "vulnerability") || strings.Contains(lower, "xss") {
		if strings.Contains(lower, "auth") || strings.Contains(lower, "jwt") {
			return "For secure JWT implementation: 1) Use RS256 algorithm, not HS256, 2) Set short expiration (15 min) for access tokens, 3) Implement refresh token rotation, 4) Store sensitive data server-side only, 5) Add rate limiting on auth endpoints. I can review your implementation."
		}
		if strings.Contains(lower, "input") || strings.Contains(lower, "validation") {
			return "Critical: Always validate and sanitize user input. Use parameterized queries to prevent SQL injection, escape HTML to prevent XSS, validate content-type headers, and implement CSP headers. Want me to review specific endpoints?"
		}
		return "From a security perspective, ensure you're following OWASP Top 10 guidelines. Priority areas: authentication, authorization, input validation, and secure communications (HTTPS/TLS). What's your main concern?"
	}

	// Generic helpful response
	return "That's an interesting question. Based on the context, I'd suggest we break this down into smaller parts. What specific aspect would you like to explore first?"
}
