package main

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/embed"
	learningpkg "github.com/camronwood/neural-junkie/internal/learning"
)

func initCodeIndexEmbed() {
	if appConfig == nil {
		return
	}
	endpoint := "http://localhost:11434"
	if p := appConfig.GetProvider("ollama-local"); p != nil && strings.TrimSpace(p.Endpoint) != "" {
		endpoint = strings.TrimRight(p.Endpoint, "/")
	}
	model := appConfig.CodebaseEmbedModel()
	if model == "" {
		model = appConfig.LearningEmbedModel()
	}
	if model == "" {
		model = embed.DefaultModel
	}
	codeindex.SetEmbedClient(embed.NewClient(endpoint, model))
	// Keep learning embed in sync
	learningpkg.SetEmbedConfig(endpoint, model)
	log.Printf("Code index embed: %s model=%s", endpoint, model)
}
