package main

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func ollamaEndpoint() string {
	endpoint := "http://localhost:11434"
	if appConfig != nil {
		if ep := appConfig.FirstOllamaEndpoint(); ep != "" {
			endpoint = ep
		}
	}
	return endpoint
}

func ollamaToolCapableTagFilter(ctx context.Context) func(string) bool {
	endpoint := ollamaEndpoint()
	return func(tag string) bool {
		return ai.OllamaTagSupportsTools(ctx, endpoint, tag)
	}
}
