package agent

import (
	"context"
	"strings"
	"testing"
)

func TestGuardWebSearchToolResult_blocksRetries(t *testing.T) {
	ctx := withWebSearchGuard(context.Background())
	first := guardWebSearchToolResult(ctx, "web_search", "ERROR: web search is not configured (enable in Settings)")
	if !strings.Contains(first, "Do not call web_search again") {
		t.Fatalf("expected guidance, got %q", first)
	}
	second := guardWebSearchToolResult(ctx, "web_search", "ERROR: web search is not configured")
	if second != first {
		t.Fatalf("expected same guidance on retry, got %q", second)
	}
}

func TestGuardWebSearchToolResult_passesOtherTools(t *testing.T) {
	ctx := withWebSearchGuard(context.Background())
	got := guardWebSearchToolResult(ctx, "fetch_url", "HTTP 200 ok")
	if got != "HTTP 200 ok" {
		t.Fatalf("unexpected rewrite: %q", got)
	}
}
