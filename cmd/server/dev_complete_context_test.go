package main

import (
	"strings"
	"testing"
)

func TestAssembleDevCompletePrompt_FIMWithNeighbors(t *testing.T) {
	prompt := AssembleDevCompletePrompt(DevCompleteRequest{
		Prefix:   "func main() {\n\tfmt.Pri",
		Suffix:   "\n}",
		Language: "go",
		Path:     "cmd/app/main.go",
		Context:  "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Pri\n}",
		NeighborSnippets: []NeighborSnippet{
			{Path: "cmd/app/util.go", Content: "package main\n\nfunc helper() {}", Source: "open_tab"},
			{Path: "internal/x.go", Content: "package internal\n", Source: "recent_edit"},
		},
	})
	if !strings.Contains(prompt, "Related open files") {
		t.Fatalf("expected neighbor section, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "cmd/app/util.go") {
		t.Fatalf("expected neighbor path, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "<|fim_prefix|>") || !strings.Contains(prompt, "<|fim_suffix|>") || !strings.Contains(prompt, "<|fim_middle|>") {
		t.Fatalf("expected FIM markers, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "fmt.Pri") {
		t.Fatalf("expected prefix in prompt, got:\n%s", prompt)
	}
}

func TestAssembleDevCompletePrompt_TruncatesLongContext(t *testing.T) {
	long := strings.Repeat("x", devCompleteMaxContextChars+5000)
	prompt := AssembleDevCompletePrompt(DevCompleteRequest{
		Prefix:  "a",
		Context: long,
		Path:    "big.go",
	})
	if len(prompt) > devCompleteMaxContextChars+devCompleteMaxNeighborChars+2000 {
		t.Fatalf("prompt too large: %d", len(prompt))
	}
	if !strings.Contains(prompt, "…") {
		t.Fatalf("expected truncation marker")
	}
}

func TestFormatNeighborSnippets_Budget(t *testing.T) {
	got := formatNeighborSnippets([]NeighborSnippet{
		{Path: "a.go", Content: strings.Repeat("A", 4000), Source: "open_tab"},
		{Path: "b.go", Content: strings.Repeat("B", 4000), Source: "open_tab"},
	}, 1500)
	if len(got) > 1600 {
		t.Fatalf("neighbors exceeded budget: %d", len(got))
	}
	if !strings.Contains(got, "a.go") {
		t.Fatalf("expected first neighbor: %s", got)
	}
}

func TestTruncatePrefixKeepsTail(t *testing.T) {
	s := "AAAA" + strings.Repeat("B", 100) + "TAIL"
	got := truncatePrefix(s, 20)
	if !strings.HasSuffix(strings.TrimSuffix(got, "\n…"), "TAIL") && !strings.Contains(got, "TAIL") {
		t.Fatalf("expected tail preserved: %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Fatalf("expected leading ellipsis: %q", got)
	}
}

func TestDedupeCompletions(t *testing.T) {
	got := dedupeCompletions([]string{"  foo  ", "foo", "", "bar", "bar"})
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Fatalf("got %#v", got)
	}
}

func TestClampDevCompleteN(t *testing.T) {
	if clampDevCompleteN(0, 2) != 2 {
		t.Fatal("default")
	}
	if clampDevCompleteN(9, 2) != 2 {
		t.Fatal("cap")
	}
	if clampDevCompleteN(1, 2) != 1 {
		t.Fatal("one")
	}
}

func TestOllamaGenerateBody(t *testing.T) {
	body := ollamaGenerateBody("m", "p", true, 128, 0.2)
	if body["stream"] != true {
		t.Fatalf("stream=%v", body["stream"])
	}
	opts := body["options"].(map[string]interface{})
	if opts["num_predict"] != 128 {
		t.Fatalf("num_predict=%v", opts["num_predict"])
	}
}
