package export

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/repo"
)

func TestBootstrapFromIndex(t *testing.T) {
	index := &repo.RepositoryIndex{
		Name:            "myapp",
		ArchitectureDoc:   strings.Repeat("Layered service architecture. ", 80),
		KeyFiles:        map[string]string{"README.md": "MyApp helps teams ship faster."},
		CodePatterns:    []string{"Go", "React"},
		SourceFiles: map[string]*repo.SourceFile{
			"internal/auth/handler.go": {
				Path:    "internal/auth/handler.go",
				Summary: "HTTP handlers for authentication.",
				Size:    4096,
			},
			"internal/db/store.go": {
				Path:    "internal/db/store.go",
				Summary: "Database access layer.",
				Size:    2048,
			},
		},
	}
	rows := BootstrapFromIndex(index)
	if len(rows) < 4 {
		t.Fatalf("expected several bootstrap rows, got %d: %#v", len(rows), rows)
	}
	kinds := map[string]int{}
	for _, r := range rows {
		if r.SourceKind != "index" {
			t.Fatalf("expected index source kind, got %q", r.SourceKind)
		}
		kinds[r.SourceRef]++
	}
	if kinds["README.md"] != 1 {
		t.Fatalf("expected README row, kinds=%v", kinds)
	}
}

func TestBootstrapFromIndexDedupes(t *testing.T) {
	text := strings.Repeat("Shared architecture overview. ", 20)
	index := &repo.RepositoryIndex{
		Name:            "dup",
		ArchitectureDoc: text,
	}
	rows := BootstrapFromIndex(index)
	if len(rows) > 3 {
		t.Fatalf("expected at most 3 architecture chunks, got %d", len(rows))
	}
}
