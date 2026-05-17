package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestOllamaModelWantsThinking(t *testing.T) {
	if !ollamaModelWantsThinking("deepseek-r1:8b") {
		t.Fatal("expected deepseek-r1 to want thinking")
	}
	if ollamaModelWantsThinking("deepseek-coder:6.7b") {
		t.Fatal("coder should not require thinking API")
	}
	if !ollamaModelWantsThinking("qwen3-thinking:latest") {
		t.Fatal("expected qwen3 thinking variant")
	}
}

func TestOllamaGenerateResponseStreamThinkingAndContent(t *testing.T) {
	body := strings.Join([]string{
		`{"message":{"thinking":"think "},"done":false}`,
		`{"message":{"thinking":"more"},"done":false}`,
		`{"message":{"content":"answer"},"done":false}`,
		`{"message":{"content":"!"},"done":true}`,
	}, "\n") + "\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Think == nil || !*req.Think {
			t.Fatal("expected think=true for deepseek-r1")
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewOllamaProviderWithConfig(srv.URL, "deepseek-r1:8b")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := p.GenerateResponseStream(ctx, "system\n---SYSTEM_PROMPT_END---\nuser q", nil)
	if err != nil {
		t.Fatalf("GenerateResponseStream: %v", err)
	}

	var thinking, content string
	for tok := range ch {
		if tok.Error != nil {
			t.Fatalf("stream error: %v", tok.Error)
		}
		thinking += tok.Thinking
		content += tok.Content
	}
	if thinking != "think more" {
		t.Fatalf("thinking = %q", thinking)
	}
	if content != "answer!" {
		t.Fatalf("content = %q", content)
	}
}

func TestOllamaHistoryRolesHumanAsUser(t *testing.T) {
	var captured OllamaRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OllamaResponse{
			Message: OllamaMessage{Role: "assistant", Content: "ok"},
			Done:    true,
		})
	}))
	defer srv.Close()

	p := NewOllamaProviderWithConfig(srv.URL, "llama3.1")
	hist := []protocol.Message{
		{
			From:    protocol.AgentInfo{Type: "human", Name: "Camron"},
			Content: "prior",
		},
	}
	_, err := p.GenerateResponse(context.Background(), "system\n---SYSTEM_PROMPT_END---\nnow", hist)
	if err != nil {
		t.Fatalf("GenerateResponse: %v", err)
	}
	foundPrior := false
	for _, m := range captured.Messages {
		if m.Role == "user" && m.Content == "prior" {
			foundPrior = true
			break
		}
	}
	if !foundPrior {
		t.Fatalf("expected human history as user role, got %+v", captured.Messages)
	}
}

func TestOllamaStreamReasoningOnlyError(t *testing.T) {
	body := `{"message":{"thinking":"only reasoning"},"done":true}` + "\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewOllamaProviderWithConfig(srv.URL, "deepseek-r1:8b")
	ch, err := p.GenerateResponseStream(context.Background(), "---SYSTEM_PROMPT_END---\nq", nil)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	var lastErr error
	for tok := range ch {
		if tok.Error != nil {
			lastErr = tok.Error
		}
	}
	if lastErr != errOllamaReasoningOnly {
		t.Fatalf("expected reasoning-only error, got %v", lastErr)
	}
}
