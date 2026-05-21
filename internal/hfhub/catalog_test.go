package hfhub

import (
	"net/http"
	"testing"
	"time"
)

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

func TestResolveDownloadRepoID(t *testing.T) {
	entry, err := FindCatalogEntry("Qwen/Qwen2.5-Coder-7B-Instruct")
	if err != nil {
		t.Fatal(err)
	}
	if got := ResolveDownloadRepoID(entry); got != "bartowski/Qwen2.5-Coder-7B-Instruct-GGUF" {
		t.Fatalf("download repo = %q", got)
	}
	if ResolveDownloadRepoID(entry) == entry.RepoID {
		t.Fatal("hosted repo_id must differ from GGUF download repo")
	}
	bio, err := FindCatalogEntry("aaditya/OpenBioLLM-Llama3-8B-GGUF")
	if err != nil {
		t.Fatal(err)
	}
	if ResolveDownloadRepoID(bio) != bio.RepoID {
		t.Fatalf("local-only entry should download from repo_id, got %q", ResolveDownloadRepoID(bio))
	}
}

func TestCatalogLocalDownloadURLsReachable(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	models, err := Library()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range models {
		if !catalogHasMode(&entry, "local") || len(entry.Files) == 0 {
			continue
		}
		hubRepo := ResolveDownloadRepoID(&entry)
		for _, f := range entry.Files {
			url := "https://huggingface.co/" + hubRepo + "/resolve/main/" + f.Filename
			req, err := http.NewRequest(http.MethodHead, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("%s: %v", entry.RepoID, err)
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				t.Fatalf("%s file %q: 404 at %s", entry.RepoID, f.Filename, hubRepo)
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound &&
				resp.StatusCode != http.StatusMovedPermanently && resp.StatusCode != http.StatusTemporaryRedirect &&
				resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s file %q: unexpected status %d", entry.RepoID, f.Filename, resp.StatusCode)
			}
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				t.Logf("skip gated check for %s (%d)", entry.RepoID, resp.StatusCode)
			}
		}
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
