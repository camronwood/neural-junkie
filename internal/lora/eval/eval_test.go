package eval

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubProvider struct {
	reply string
}

func (s stubProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	return s.reply, nil
}

func (s stubProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	return s.reply, nil
}

func (s stubProvider) GetModel() string { return "stub" }

func TestScoreQuestionRubric(t *testing.T) {
	q := Question{
		Mode:           "rubric",
		ExpectedPoints: []string{"SELECT", "FROM", "users"},
		Keywords:       []string{"select"},
	}
	got := scoreQuestion("Use SELECT * FROM users WHERE created_at > now()", q)
	if got != 1.0 {
		t.Fatalf("expected 1.0, got %v", got)
	}
	got = scoreQuestion("talk about users only", q)
	if got <= 0 || got >= 1 {
		t.Fatalf("expected partial score, got %v", got)
	}
}

func TestScoreQuestionToolProbe(t *testing.T) {
	q := Question{
		Keywords:  []string{"docker", "compose"},
		ToolProbe: true,
	}
	short := scoreQuestion("use docker", q)
	if short != 0 {
		t.Fatalf("short reply should fail tool probe, got %v", short)
	}
	long := "You should run docker compose up -d to start the stack for local development."
	got := scoreQuestion(long, q)
	if got <= 0 {
		t.Fatalf("expected tool probe hit, got %v", got)
	}
}

func TestRunEmptyQuestionsCompat(t *testing.T) {
	res := Run(context.Background(), nil, "", nil, 0)
	if !res.Passed || res.Score != 1 || res.MinScore != DefaultMinScore {
		t.Fatalf("empty questions compat: %+v", res)
	}
}

func TestRunUsesMinScore(t *testing.T) {
	p := stubProvider{reply: "SELECT FROM users WHERE date"}
	qs := []Question{{
		Prompt:         "sql?",
		Mode:           "rubric",
		ExpectedPoints: []string{"SELECT", "FROM", "users", "missing"},
	}}
	res := Run(context.Background(), p, "m", qs, 0.9)
	if res.Passed {
		t.Fatalf("expected fail under high minScore, got %+v", res)
	}
	if res.MinScore != 0.9 {
		t.Fatalf("minScore: %v", res.MinScore)
	}
}
