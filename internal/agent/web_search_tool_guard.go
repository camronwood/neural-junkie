package agent

import (
	"context"
	"strings"
)

type webSearchGuard struct {
	blocked bool
}

type webSearchGuardKey struct{}

const webSearchUnavailableGuidance = "Web search is not configured. Do not call web_search again this turn. " +
	"Tell the user to enable Settings → Integrations → Web search (Tavily or Brave), " +
	"then answer from general knowledge with appropriate caveats for time-sensitive facts."

func withWebSearchGuard(ctx context.Context) context.Context {
	if webSearchGuardFrom(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, webSearchGuardKey{}, &webSearchGuard{})
}

func webSearchGuardFrom(ctx context.Context) *webSearchGuard {
	if ctx == nil {
		return nil
	}
	g, _ := ctx.Value(webSearchGuardKey{}).(*webSearchGuard)
	return g
}

func toolResultWebSearchNotConfigured(result string) bool {
	return strings.Contains(strings.ToLower(result), "web search is not configured")
}

func guardWebSearchToolResult(ctx context.Context, toolName, result string) string {
	if toolName != "web_search" {
		return result
	}
	g := webSearchGuardFrom(ctx)
	if g != nil && g.blocked {
		return webSearchUnavailableGuidance
	}
	if toolResultWebSearchNotConfigured(result) {
		if g != nil {
			g.blocked = true
		}
		return webSearchUnavailableGuidance
	}
	return result
}
