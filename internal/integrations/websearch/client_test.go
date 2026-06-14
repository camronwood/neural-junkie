package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestSearchBrave(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Subscription-Token"); got != "test-key" {
			t.Fatalf("token = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"web": map[string]interface{}{
				"results": []map[string]string{
					{"title": "Neural Junkie", "url": "https://example.com/nj", "description": "Multi-agent dev tool"},
				},
			},
		})
	}))
	defer srv.Close()

	old := braveSearchBaseURL
	braveSearchBaseURL = srv.URL
	t.Cleanup(func() { braveSearchBaseURL = old })

	client := NewClient(config.WebSearchConfig{
		Enabled:  true,
		Provider: "brave",
		APIKey:   "test-key",
	})
	results, err := client.Search(context.Background(), "neural junkie", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].URL != "https://example.com/nj" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestSearchTavily(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tvly-test" {
			t.Fatalf("auth = %q", got)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != "golang release" {
			t.Fatalf("query = %v", body["query"])
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]string{
				{"title": "Go", "url": "https://go.dev", "content": "The Go programming language"},
			},
		})
	}))
	defer srv.Close()

	old := tavilySearchBaseURL
	tavilySearchBaseURL = srv.URL
	t.Cleanup(func() { tavilySearchBaseURL = old })

	client := NewClient(config.WebSearchConfig{
		Enabled:  true,
		Provider: "tavily",
		APIKey:   "tvly-test",
	})
	results, err := client.Search(context.Background(), "golang release", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Description == "" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestSearchTavilyKeyless(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Tavily-Access-Mode"); got != "keyless" {
			t.Fatalf("keyless header = %q", got)
		}
		if r.Header.Get("Authorization") != "" {
			t.Fatal("expected no Authorization header in keyless mode")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]string{
				{"title": "Example", "url": "https://example.com", "content": "snippet"},
			},
		})
	}))
	defer srv.Close()

	old := tavilySearchBaseURL
	tavilySearchBaseURL = srv.URL
	t.Cleanup(func() { tavilySearchBaseURL = old })

	client := NewClient(config.WebSearchConfig{
		Enabled:  true,
		Provider: "tavily",
		Keyless:  true,
	})
	if !client.Ready() {
		t.Fatal("expected keyless tavily client to be ready")
	}
	results, err := client.Search(context.Background(), "test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v", results)
	}
}

func TestSearchRequiresConfig(t *testing.T) {
	client := NewClient(config.WebSearchConfig{})
	_, err := client.Search(context.Background(), "test", 1)
	if err == nil {
		t.Fatal("expected error when not configured")
	}
}
