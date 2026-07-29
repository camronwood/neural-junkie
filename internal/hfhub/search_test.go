package hfhub

import (
	"strings"
	"testing"
)

func TestTitleFromRepoID(t *testing.T) {
	if got := titleFromRepoID("Qwen/Qwen2.5-Coder-7B-Instruct"); got == "" {
		t.Fatal("expected title")
	}
}

func TestQuantFromFilename(t *testing.T) {
	if got := quantFromFilename("model-Q4_K_M.gguf"); got != "Q4_K_M" {
		t.Fatalf("quant = %q", got)
	}
}

func TestResolveDownloadTargetCatalog(t *testing.T) {
	repo, fn, err := ResolveDownloadTarget("Qwen/Qwen2.5-Coder-7B-Instruct", "")
	if err != nil {
		t.Fatal(err)
	}
	if repo != "bartowski/Qwen2.5-Coder-7B-Instruct-GGUF" {
		t.Fatalf("download repo = %q", repo)
	}
	if fn == "" {
		t.Fatal("expected default filename")
	}
}

func TestResolveDownloadTargetAdHoc(t *testing.T) {
	repo, fn, err := ResolveDownloadTarget("org/custom-gguf", "weights/model.Q4_K_M.gguf")
	if err != nil {
		t.Fatal(err)
	}
	if repo != "org/custom-gguf" || fn != "weights/model.Q4_K_M.gguf" {
		t.Fatalf("repo=%q fn=%q", repo, fn)
	}
}

func TestResolveDownloadTargetAdHocRequiresFilename(t *testing.T) {
	_, _, err := ResolveDownloadTarget("org/custom-gguf", "")
	if err == nil {
		t.Fatal("expected error without filename")
	}
}

func TestMergeCatalogWithSearch(t *testing.T) {
	curated := []LibraryModel{
		{RepoID: "Qwen/Qwen2.5-Coder-7B-Instruct", Modes: []string{"hosted", "local"}},
	}
	hits := []SearchHit{
		{RepoID: "Qwen/Qwen2.5-Coder-7B-Instruct", Title: "dup"},
		{RepoID: "meta-llama/Llama-3.1-8B-Instruct", Title: "Llama", Modes: []string{"hosted"}},
	}
	merged := MergeCatalogWithSearch(curated, hits, "hosted")
	if len(merged) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(merged))
	}
	if merged[1].RepoID != "meta-llama/Llama-3.1-8B-Instruct" {
		t.Fatalf("unexpected second row: %#v", merged[1])
	}
}

func TestPreferCommonQuants(t *testing.T) {
	files := []CatalogFile{
		{Filename: "a.Q2_K.gguf", Quant: "Q2_K"},
		{Filename: "b.Q4_K_M.gguf", Quant: "Q4_K_M"},
	}
	out := preferCommonQuants(files)
	if !strings.Contains(out[0].Filename, "Q4_K_M") {
		t.Fatalf("expected Q4_K_M first, got %#v", out)
	}
}

func TestEscapeHFRepoPath(t *testing.T) {
	got := escapeHFRepoPath("GrEarl/Kimi-K3-GGUF")
	if got != "GrEarl/Kimi-K3-GGUF" {
		t.Fatalf("got %q, want unencoded slash", got)
	}
	if strings.Contains(got, "%2F") || strings.Contains(got, "%2f") {
		t.Fatalf("must not url-encode slash: %q", got)
	}
	got = escapeHFRepoPath("org/model with space")
	if got != "org/model%20with%20space" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatSizeHint(t *testing.T) {
	if got := formatSizeHint(0); got != "" {
		t.Fatalf("empty for 0, got %q", got)
	}
	if got := formatSizeHint(668_203_584); got != "~637 MB" {
		t.Fatalf("got %q want ~637 MB", got)
	}
	if got := formatSizeHint(4_700_000_000); got != "~4.4 GB" {
		t.Fatalf("got %q want ~4.4 GB", got)
	}
	if got := formatSizeHint(12 * 1024 * 1024 * 1024); got != "~12 GB" {
		t.Fatalf("got %q want ~12 GB", got)
	}
}
