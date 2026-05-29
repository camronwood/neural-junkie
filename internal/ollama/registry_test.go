package ollama

import "testing"

const sampleLibraryHTML = `
<a href="/library/qwen2.5-coder">Qwen2.5 Coder</a>
<a href="/library/qwen2.5-coder:7b">bad</a>
<a href="/library/llama3.1">Llama 3.1</a>
<a href="/library/qwen2.5-coder">dup</a>
`

const sampleTagsHTML = `
<a href="/library/qwen2.5-coder:latest">latest</a>
<a href="/library/qwen2.5-coder:7b">7b</a>
<a href="/library/qwen2.5-coder:7b-base-q4_K_M">quant</a>
<a href="/library/qwen2.5-coder:14b">14b</a>
`

func TestParseLibraryNames(t *testing.T) {
	names := uniqueStrings(parseLibraryNames(sampleLibraryHTML))
	if len(names) != 2 {
		t.Fatalf("expected 2 unique names, got %d: %#v", len(names), names)
	}
	if names[0] != "qwen2.5-coder" || names[1] != "llama3.1" {
		t.Fatalf("unexpected order: %#v", names)
	}
}

func TestParseTagNames(t *testing.T) {
	tags := parseTagNames(sampleTagsHTML, "qwen2.5-coder")
	if len(tags) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(tags))
	}
}

func TestFilterCommonTags(t *testing.T) {
	filtered := filterCommonTags("qwen2.5-coder", []string{
		"qwen2.5-coder:latest",
		"qwen2.5-coder:7b-base-q4_K_M",
		"qwen2.5-coder:7b",
		"qwen2.5-coder:14b",
	})
	if len(filtered) != 3 {
		t.Fatalf("expected 3 common tags, got %#v", filtered)
	}
}

func TestMergeCatalogWithRegistry(t *testing.T) {
	curated := []LibraryModel{
		{Name: "qwen2.5-coder:14b", Title: "Curated Qwen", Tags: []string{"recommended"}},
	}
	registry := []RegistryModel{
		{Name: "qwen2.5-coder", Title: "Registry Qwen"},
		{Name: "mistral", Title: "Mistral"},
	}
	merged := MergeCatalogWithRegistry(curated, registry)
	if len(merged) != 2 {
		t.Fatalf("expected curated + one new registry row, got %d: %#v", len(merged), merged)
	}
	if merged[0].Title != "Curated Qwen" {
		t.Fatalf("curated row should stay first with metadata: %#v", merged[0])
	}
	if merged[1].Name != "mistral" {
		t.Fatalf("expected mistral appended, got %#v", merged[1])
	}
}

func TestNormalizeRegistryName(t *testing.T) {
	if got := normalizeRegistryName("https://ollama.com/library/qwen2.5-coder:7b"); got != "qwen2.5-coder" {
		t.Fatalf("normalize = %q", got)
	}
}
