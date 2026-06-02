package embed

import (
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{1, 0, 0}
	if CosineSimilarity(a, b) < 0.99 {
		t.Fatal("identical vectors should score ~1")
	}
	c := []float64{0, 1, 0}
	if CosineSimilarity(a, c) > 0.01 {
		t.Fatal("orthogonal vectors should score ~0")
	}
}

func TestKeywordScore(t *testing.T) {
	score := KeywordScore("please use tabs in go", "Always use tabs for indentation in Go files")
	if score <= 0 {
		t.Fatal("expected positive keyword overlap")
	}
}
