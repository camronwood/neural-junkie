package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func TestClassifyUserFacingErrorOllamaModelMissing(t *testing.T) {
	err := fmt.Errorf(`Ollama API request failed with status 404: {"error":"model 'nj-bio:8b' not found"}`)
	msg, code, retryable := classifyUserFacingError(err)
	if code != "provider_unavailable" {
		t.Fatalf("code=%s", code)
	}
	if retryable {
		t.Fatal("expected not retryable")
	}
	if !containsAll(msg, "nj-bio:8b", "Model Library") {
		t.Fatalf("message=%q", msg)
	}
}

func TestClassifyUserFacingErrorOllamaNoContent(t *testing.T) {
	msg, code, retryable := classifyUserFacingError(ai.ErrOllamaNoContent)
	if code != "provider_error" || !retryable {
		t.Fatalf("code=%s retryable=%v", code, retryable)
	}
	if !containsAll(msg, "empty reply", "nj-bio") {
		t.Fatalf("message=%q", msg)
	}
}

func TestClassifyUserFacingErrorCLIExecutable(t *testing.T) {
	err := fmt.Errorf("exec: \"agent\": executable file not found in $PATH")
	msg, code, _ := classifyUserFacingError(err)
	if code != "provider_unavailable" {
		t.Fatalf("code=%s", code)
	}
	if !containsAll(msg, "CLI agent") {
		t.Fatalf("message=%q", msg)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
