package hfhub

import "testing"

func TestLibraryParses(t *testing.T) {
	models, err := Library()
	if err != nil {
		t.Fatal(err)
	}
	if len(models) < 5 {
		t.Fatalf("expected at least 5 catalog entries, got %d", len(models))
	}
}

func TestFindCatalogEntry(t *testing.T) {
	_, err := FindCatalogEntry("Qwen/Qwen2.5-Coder-7B-Instruct")
	if err != nil {
		t.Fatal(err)
	}
	_, err = FindCatalogEntry("not/in/catalog")
	if err == nil {
		t.Fatal("expected error for unknown repo")
	}
}

func TestResolveDownloadFilename(t *testing.T) {
	entry, err := FindCatalogEntry("Qwen/Qwen2.5-Coder-7B-Instruct")
	if err != nil {
		t.Fatal(err)
	}
	fn, err := ResolveDownloadFilename(entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if fn == "" {
		t.Fatal("expected default filename")
	}
}

func TestBioGGUFDefaultFilename(t *testing.T) {
	entry, err := FindCatalogEntry("aaditya/OpenBioLLM-Llama3-8B-GGUF")
	if err != nil {
		t.Fatal(err)
	}
	fn, err := ResolveDownloadFilename(entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if fn != "openbiollm-llama3-8b.Q4_K_M.gguf" {
		t.Fatalf("filename = %q (must match Hub repo paths)", fn)
	}
}
