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

const sampleXTagsHTML = `
<a href="/x/flux2-klein:latest">latest</a>
<a href="/x/flux2-klein:4b">4b</a>
<a href="/x/flux2-klein:9b">9b</a>
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

func TestParseLibraryNamesIncludesXNamespace(t *testing.T) {
	html := `<a href="/x/flux2-klein">FLUX</a><a href="/library/llama3.1">Llama</a>`
	names := uniqueStrings(parseLibraryNames(html))
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %#v", names)
	}
	seen := make(map[string]bool)
	for _, n := range names {
		seen[n] = true
	}
	if !seen["x/flux2-klein"] || !seen["llama3.1"] {
		t.Fatalf("unexpected: %#v", names)
	}
}

func TestParseTagNames(t *testing.T) {
	tags := parseTagNames(sampleTagsHTML, "qwen2.5-coder")
	if len(tags) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(tags))
	}
}

func TestParseTagNamesXNamespace(t *testing.T) {
	tags := parseTagNames(sampleXTagsHTML, "x/flux2-klein")
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %#v", tags)
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
	if got := normalizeRegistryName("https://ollama.com/x/flux2-klein:4b"); got != "x/flux2-klein" {
		t.Fatalf("normalize x namespace = %q", got)
	}
}

func TestRegistryTagsPageURL(t *testing.T) {
	if got := registryTagsPageURL("qwen2.5-coder"); got != "https://ollama.com/library/qwen2.5-coder/tags" {
		t.Fatalf("library tags url = %q", got)
	}
	if got := registryTagsPageURL("x/flux2-klein"); got != "https://ollama.com/x/flux2-klein/tags" {
		t.Fatalf("x namespace tags url = %q", got)
	}
}

func TestParseTagSizeHints(t *testing.T) {
	html := `</span> <p class="flex text-neutral-500">4.9GB · 128K context window · Text · 1 year ago</p>`
	// weaker page without tag suffix — parseFirstSizeHint
	if got := parseFirstSizeHint(html); got != "~4.9 GB" {
		t.Fatalf("parseFirstSizeHint = %q", got)
	}
	html2 := `<p class="truncate hover:underline text-sm font-medium text-neutral-800">llama3.1:latest</p> </span> <p class="flex text-neutral-500">4.9GB · 128K</p>`
	m := parseTagSizeHints(html2)
	if m["latest"] != "~4.9 GB" {
		t.Fatalf("latest size = %#v", m)
	}
}
