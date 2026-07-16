package eval

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
)

// Question is one golden Q&A probe.
type Question struct {
	Prompt         string   `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Keywords       []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Mode           string   `json:"mode,omitempty" yaml:"mode,omitempty"` // "keywords" (default) or "rubric"
	ExpectedPoints []string `json:"expected_points,omitempty" yaml:"expected_points,omitempty"`
	ToolProbe      bool     `json:"tool_probe,omitempty" yaml:"tool_probe,omitempty"`
}

// Result is a post-train eval score.
type Result struct {
	Score    float64  `json:"score"`
	Passed   bool     `json:"passed"`
	Details  []string `json:"details,omitempty"`
	MinScore float64  `json:"min_score"`
}

// DefaultMinScore is the pass threshold when callers pass minScore <= 0.
const DefaultMinScore = 0.70

// Run executes keyword/rubric overlap eval against composed model output.
// Empty questions keep Score:1 Passed:true for backward compatibility when no
// probes were configured at assign time (handlers short-circuit empty lists too).
// If minScore <= 0, DefaultMinScore is used.
func Run(ctx context.Context, provider ai.AIProvider, model string, questions []Question, minScore float64) Result {
	if minScore <= 0 {
		minScore = DefaultMinScore
	}
	if len(questions) == 0 {
		return Result{Score: 1, Passed: true, MinScore: minScore}
	}
	if provider == nil {
		return Result{Score: 0, Passed: false, MinScore: minScore, Details: []string{"no provider"}}
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
		ratio := scoreQuestion(reply, q)
		hits += ratio
		details = append(details, strings.TrimSpace(q.Prompt[:min(40, len(q.Prompt))])+"...")
	}
	n := float64(len(questions))
	if n == 0 {
		return Result{Score: 1, Passed: true, MinScore: minScore}
	}
	score := hits / n
	return Result{
		Score:    score,
		Passed:   score >= minScore,
		Details:  details,
		MinScore: minScore,
	}
}

func scoreQuestion(reply string, q Question) float64 {
	text := strings.ToLower(reply)
	useRubric := strings.EqualFold(strings.TrimSpace(q.Mode), "rubric") || len(q.ExpectedPoints) > 0

	ratio := 0.0
	if useRubric && len(q.ExpectedPoints) > 0 {
		matched := 0
		for _, point := range q.ExpectedPoints {
			if point == "" {
				continue
			}
			if strings.Contains(text, strings.ToLower(point)) {
				matched++
			}
		}
		ratio = float64(matched) / float64(len(q.ExpectedPoints))
	} else {
		matched := 0
		for _, kw := range q.Keywords {
			if kw == "" {
				continue
			}
			if strings.Contains(text, strings.ToLower(kw)) {
				matched++
			}
		}
		if len(q.Keywords) == 0 {
			if len(strings.TrimSpace(reply)) > 20 {
				matched = 1
			}
		}
		if len(q.Keywords) > 0 {
			ratio = float64(matched) / float64(len(q.Keywords))
		} else if matched > 0 {
			ratio = 1
		}
	}

	if q.ToolProbe {
		if len(reply) <= 40 || !toolProbeHit(text, q) {
			return 0
		}
	}
	return ratio
}

func toolProbeHit(textLower string, q Question) bool {
	for _, term := range q.ExpectedPoints {
		if term != "" && strings.Contains(textLower, strings.ToLower(term)) {
			return true
		}
	}
	for _, term := range q.Keywords {
		if term != "" && strings.Contains(textLower, strings.ToLower(term)) {
			return true
		}
	}
	return false
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
