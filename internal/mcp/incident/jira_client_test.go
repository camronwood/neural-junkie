package incident

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestClientGetMyself(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Fatal("missing basic auth")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountId":"abc","displayName":"Test User"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(config.JiraConfig{
		BaseURL:   srv.URL,
		Email:     "user@example.com",
		APIToken:  "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	client.HTTP = srv.Client()

	out, err := client.GetMyself(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Test User") {
		t.Fatalf("unexpected response: %s", out)
	}
}

func TestSummarizeIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/3/issue/ENG-1") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"key":"ENG-1",
			"fields":{
				"summary":"Login fails",
				"status":{"name":"Open"},
				"priority":{"name":"High"},
				"labels":["regression"],
				"description":"Steps to reproduce"
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(config.JiraConfig{
		BaseURL:  srv.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	client.HTTP = srv.Client()

	out, err := client.SummarizeIssue(context.Background(), "ENG-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ENG-1", "Login fails", "Open", "High", "regression"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary missing %q: %s", want, out)
		}
	}
}
