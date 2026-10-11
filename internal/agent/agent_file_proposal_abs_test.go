package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSanitizeAbsolutePathFileChange_osAgnostic(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"unix_users", `[FILE_CHANGE path="/Users/me/proj/a.go"]`},
		{"unix_root", `[FILE_CHANGE path="/tmp/proj/a.go"]`},
		{"windows_drive", `[FILE_CHANGE path="C:\Users\me\proj\a.go"]`},
		{"windows_slash", `[FILE_CHANGE path="D:/work/proj/a.go"]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := sanitizeAbsolutePathFileChangeFromResponse("hello " + tc.in + " world")
			if strings.Contains(out, "FILE_CHANGE") {
				t.Fatalf("absolute path not stripped: %q", out)
			}
			if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
				t.Fatalf("surrounding text damaged: %q", out)
			}
		})
	}
	rel := `[FILE_CHANGE path="src/a.go"] keep`
	if got := sanitizeAbsolutePathFileChangeFromResponse(rel); !strings.Contains(got, "FILE_CHANGE") {
		t.Fatalf("relative path must be kept: %q", got)
	}
}

func TestIsOSAbsolutePath(t *testing.T) {
	if !isOSAbsolutePath("/tmp/x") {
		t.Fatal("unix abs")
	}
	if !isOSAbsolutePath(`C:\tmp\x`) {
		t.Fatal("windows abs")
	}
	if isOSAbsolutePath("src/main.go") {
		t.Fatal("relative must be false")
	}
}

func TestSubmitAllFileChangesFromResponse_multiBlock(t *testing.T) {
	a := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	dir := t.TempDir()
	a.WorkspacePath = dir
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "u", Type: "human"}, "create two files")
	msg.Metadata = map[string]interface{}{
		protocol.IdeMetaImplementationSession: true,
		"editor_mode":                         "agent",
		"editor_agent_trust":                  editorTrustAutoApply,
		"workspace_context":                   map[string]interface{}{"workspace_path": dir},
	}
	resp := "" +
		"[FILE_CHANGE]\noperation: create\npath: one.txt\n```\nalpha\n```\n[/FILE_CHANGE]\n" +
		"[FILE_CHANGE]\noperation: create\npath: two.txt\n```\nbeta\n```\n[/FILE_CHANGE]\n"
	cleaned, ok, err := a.maybeSubmitFileChangeFromResponse(context.Background(), resp, msg.Channel, msg)
	if err != nil {
		t.Fatalf("submit err: %v", err)
	}
	if !ok {
		t.Fatal("expected proposals submitted")
	}
	if strings.Contains(cleaned, "[FILE_CHANGE]") {
		t.Fatalf("blocks should be stripped: %q", cleaned)
	}
}
