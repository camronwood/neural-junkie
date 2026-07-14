package hfhub

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func catalogNetworkChecksEnabled() bool {
	if testing.Short() {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("HF_CATALOG_NETWORK"))) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func isTransientNetworkErr(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "tls handshake") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}

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
	if !catalogNetworkChecksEnabled() {
		t.Skip("HF catalog network checks disabled (short mode or HF_CATALOG_NETWORK=0)")
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
				if isTransientNetworkErr(err) {
					t.Skipf("skip transient network error for %s: %v", entry.RepoID, err)
				}
				t.Fatalf("%s: %v", entry.RepoID, err)
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				t.Fatalf("%s file %q: 404 at %s", entry.RepoID, f.Filename, hubRepo)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				t.Logf("skip rate-limited check for %s (%d)", entry.RepoID, resp.StatusCode)
				continue
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

func TestOrnithGGUFDefaultFilename(t *testing.T) {
	entry, err := FindCatalogEntry("deepreinforce-ai/Ornith-1.0-9B")
	if err != nil {
		t.Fatal(err)
	}
	if got := ResolveDownloadRepoID(entry); got != "deepreinforce-ai/Ornith-1.0-9B-GGUF" {
		t.Fatalf("download repo = %q", got)
	}
	fn, err := ResolveDownloadFilename(entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if fn != "ornith-1.0-9b-Q4_K_M.gguf" {
		t.Fatalf("filename = %q (must match Hub repo paths)", fn)
	}
}

func TestBonsaiGGUFPrimaryAndMmproj(t *testing.T) {
	entry, err := FindCatalogEntry("prism-ml/Bonsai-27B-gguf")
	if err != nil {
		t.Fatal(err)
	}
	fn, err := ResolveDownloadFilename(entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if fn != "Bonsai-27B-Q1_0.gguf" {
		t.Fatalf("primary = %q", fn)
	}
	mm := MmprojCatalogFile(entry)
	if mm == nil || mm.Filename != "Bonsai-27B-mmproj-Q8_0.gguf" {
		t.Fatalf("mmproj = %#v", mm)
	}
	if entry.DefaultOllamaTag != "nj-bonsai:27b" {
		t.Fatalf("default tag = %q", entry.DefaultOllamaTag)
	}
}

func TestTernaryBonsaiGGUFPrimary(t *testing.T) {
	entry, err := FindCatalogEntry("prism-ml/Ternary-Bonsai-27B-gguf")
	if err != nil {
		t.Fatal(err)
	}
	fn, err := ResolveDownloadFilename(entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if fn != "Ternary-Bonsai-27B-Q2_0.gguf" {
		t.Fatalf("primary = %q", fn)
	}
}

func TestResolveDownloadFilenameLoRACompanion(t *testing.T) {
	entry, err := FindCatalogEntry("scthornton/llama-3.2-3b-securecode")
	if err != nil {
		t.Fatal(err)
	}
	fn, err := ResolveDownloadFilename(entry, AdapterConfigFilename)
	if err != nil {
		t.Fatal(err)
	}
	if fn != AdapterConfigFilename {
		t.Fatalf("filename = %q", fn)
	}
	_, err = ResolveDownloadFilename(entry, "not-a-real-file.bin")
	if err == nil {
		t.Fatal("expected error for unknown filename")
	}
}

func TestAdapterOllamaComposeSupported(t *testing.T) {
	llama, err := FindCatalogEntry("scthornton/llama-3.2-3b-securecode")
	if err != nil {
		t.Fatal(err)
	}
	if !AdapterOllamaComposeSupported(llama) {
		t.Fatal("expected llama securecode compose supported")
	}
	qwen, err := FindCatalogEntry("scthornton/qwen2.5-coder-14b-securecode")
	if err != nil {
		t.Fatal(err)
	}
	if AdapterOllamaComposeSupported(qwen) {
		t.Fatal("expected deprecated qwen adapter unsupported")
	}
}

func TestDefaultLoRABaseTag(t *testing.T) {
	if DefaultLoRABaseTag != DefaultLoRATrainingCodeBase {
		t.Fatalf("DefaultLoRABaseTag = %q, want %q", DefaultLoRABaseTag, DefaultLoRATrainingCodeBase)
	}
	if DefaultLoRABaseTag != "llama3.1:8b" {
		t.Fatalf("DefaultLoRABaseTag = %q", DefaultLoRABaseTag)
	}
}
