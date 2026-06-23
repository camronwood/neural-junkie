package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeSubmitGitChangeFromResponse(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "test", ID: "a1"}}
	response := `Done.
[GIT_CHANGE]
operation: commit
message: fix tests
paths: foo.go, bar.go
[/GIT_CHANGE]`
	stripped, proposed, err := a.maybeSubmitGitChangeFromResponse(context.Background(), response, "general", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proposed {
		t.Fatal("expected no proposal without hub")
	}
	if stripped == response {
		t.Fatal("expected git block stripped from response")
	}
}

func TestStripGitChangeBlocks(t *testing.T) {
	in := "ok\n[GIT_CHANGE]\noperation: push\n[/GIT_CHANGE]"
	out := stripGitChangeBlocks(in)
	if out != "ok" {
		t.Fatalf("got %q", out)
	}
}
