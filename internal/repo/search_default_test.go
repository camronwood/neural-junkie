package repo

import "testing"

func TestSearchRelevantFilesFallbackWhenNoKeywordMatch(t *testing.T) {
	content, _, err := CompressContent("package main\nfunc main() {}")
	if err != nil {
		t.Fatal(err)
	}
	index := &RepositoryIndex{
		SourceFiles: map[string]*SourceFile{
			"internal/obscure/x.go": {Path: "internal/obscure/x.go", Language: "Go", Content: content},
			"main.go":               {Path: "main.go", Language: "Go", Content: content},
		},
	}
	files := SearchRelevantFiles("hello there", index, 3)
	if len(files) == 0 {
		t.Fatal("expected fallback files when keywords do not match paths")
	}
	foundMain := false
	for _, f := range files {
		if f.Path == "main.go" {
			foundMain = true
		}
	}
	if !foundMain {
		t.Fatalf("expected main.go in fallback results, got %d files", len(files))
	}
}
