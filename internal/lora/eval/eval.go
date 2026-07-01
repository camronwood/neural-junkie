package eval

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
)

// Question is one golden Q&A probe.
type Question struct {
	Prompt   string
	Keywords []string
}

// Result is a post-train eval score.
type Result struct {
	Score    float64  `json:"score"`
	Passed   bool     `json:"passed"`
	Details  []string `json:"details,omitempty"`
	MinScore float64  `json:"min_score"`
}

const DefaultMinScore = 0.35

// Run executes keyword-overlap eval against composed model output.
func Run(ctx context.Context, provider ai.AIProvider, model string, questions []Question) Result {
	if len(questions) == 0 {
		return Result{Score: 1, Passed: true, MinScore: DefaultMinScore}
	}
	if provider == nil {
		return Result{Score: 0, Passed: false, MinScore: DefaultMinScore, Details: []string{"no provider"}}
	}
	var hits float64
	var details []string
	for _, q := range questions {
		if strings.TrimSpace(q.Prompt) == "" {
			continue
		}
		reply, err := provider.GenerateResponse(ctx, q.Prompt, nil)
		if err != nil {
			details = append(details, "error: "+err.Error())
			continue
		}
		text := strings.ToLower(reply)
		matched := 0
		for _, kw := range q.Keywords {
			if strings.Contains(text, strings.ToLower(kw)) {
				matched++
			}
		}
		if len(q.Keywords) == 0 {
			if len(strings.TrimSpace(reply)) > 20 {
				matched = 1
			}
		}
		ratio := 0.0
		if len(q.Keywords) > 0 {
			ratio = float64(matched) / float64(len(q.Keywords))
		} else if matched > 0 {
			ratio = 1
		}
		hits += ratio
		details = append(details, strings.TrimSpace(q.Prompt[:min(40, len(q.Prompt))])+"...")
	}
	score := hits / float64(len(questions))
	return Result{
		Score:    score,
		Passed:   score >= DefaultMinScore,
		Details:  details,
		MinScore: DefaultMinScore,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DefaultRepoQuestions returns generic repo eval probes.
func DefaultRepoQuestions(repoName string) []Question {
	name := strings.TrimSpace(repoName)
	if name == "" {
		name = "this repository"
	}
	return []Question{
		{Prompt: "What is the primary language or stack used in " + name + "?", Keywords: []string{"go", "python", "typescript", "rust", "java"}},
		{Prompt: "Summarize the architecture of " + name + " in one sentence.", Keywords: []string{"component", "service", "module", "layer"}},
		{Prompt: "What testing approach does " + name + " use?", Keywords: []string{"test", "unit", "integration"}},
	}
}
