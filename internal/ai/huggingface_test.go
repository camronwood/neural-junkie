package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestHuggingFaceProviderGenerateResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("auth header %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "hi"}},
			},
		})
	}))
	defer srv.Close()

	p := NewHuggingFaceProvider(srv.URL+"/v1", "test-token", "org/model")
	out, err := p.GenerateResponse(context.Background(), "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi" {
		t.Fatalf("got %q", out)
	}
}

func TestProviderFromConfigHuggingface(t *testing.T) {
	_, err := ProviderFromConfig(&config.ProviderConfig{
		ID: "hf", Type: "huggingface", APIKey: "tok", Model: "Qwen/Qwen2.5-7B-Instruct",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ProviderFromConfig(&config.ProviderConfig{ID: "hf", Type: "huggingface"})
	if err == nil {
		t.Fatal("expected error without model")
	}
}
