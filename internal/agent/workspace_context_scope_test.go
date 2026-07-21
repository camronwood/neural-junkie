package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestResolveContextScope(t *testing.T) {
	t.Run("explicit none", func(t *testing.T) {
		msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "u"}, "hi")
		msg.Metadata = map[string]interface{}{
			MetadataContextScope: ContextScopeNone,
			"workspace_context":  map[string]interface{}{"workspace_path": "/x"},
		}
		if got := ResolveContextScope(msg); got != ContextScopeNone {
			t.Fatalf("got %q want none", got)
		}
	})
	t.Run("legacy full", func(t *testing.T) {
		msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "u"}, "hi")
		msg.Metadata = map[string]interface{}{
			"workspace_context": map[string]interface{}{"workspace_path": "/x"},
		}
		if got := ResolveContextScope(msg); got != ContextScopeFull {
			t.Fatalf("got %q want full", got)
		}
	})
	t.Run("absent", func(t *testing.T) {
		msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "u"}, "hi")
		if got := ResolveContextScope(msg); got != ContextScopeNone {
			t.Fatalf("got %q want none", got)
		}
	})
}

func TestAppendWorkspaceContext_ScopeTiers(t *testing.T) {
	baseCtx := map[string]interface{}{
		"workspace_name": "Proj",
		"workspace_path": "/proj",
		"file_tree":      "src/\n  main.go",
		"open_files": []interface{}{
			map[string]interface{}{
				"path": "src/main.go", "language": "go", "content": "package main\n", "is_active": true,
			},
		},
	}

	cases := []struct {
		scope     string
		wantTree  bool
		wantFiles bool
		wantHint  bool
		wantEmpty bool
	}{
		{ContextScopeNone, false, false, false, true},
		{ContextScopeHint, false, false, true, false},
		{ContextScopeOutline, true, false, false, false},
		{ContextScopeFocus, true, true, false, false},
		{ContextScopeFull, true, true, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.scope, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "u"}, "q")
			msg.Metadata = map[string]interface{}{
				MetadataContextScope: tc.scope,
				"workspace_context":  baseCtx,
			}
			var b strings.Builder
			AppendWorkspaceContext(&b, msg)
			out := b.String()
			if tc.wantEmpty {
				if out != "" {
					t.Fatalf("expected empty prompt, got %q", out)
				}
				return
			}
			if !strings.Contains(out, "WORKSPACE CONTEXT") {
				t.Fatal("missing workspace section")
			}
			if tc.wantHint && !strings.Contains(out, "NOT shared file contents") {
				t.Fatal("hint framing expected")
			}
			if tc.wantTree != strings.Contains(out, "file tree") {
				t.Fatalf("wantTree=%v tree in output=%v", tc.wantTree, strings.Contains(out, "file tree"))
			}
			if tc.wantFiles != strings.Contains(out, "Open files") {
				t.Fatalf("wantFiles=%v files in output=%v", tc.wantFiles, strings.Contains(out, "Open files"))
			}
		})
	}
}

func TestAppendWorkspaceContextForChannel_RetrimsFullPayloadToFocus(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-softwarearchitect",
		protocol.AgentInfo{Name: "u"},
		"review src/config.go and the active file",
	)
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeFull,
		"workspace_context": map[string]interface{}{
			"workspace_name": "Proj",
			"workspace_path": "/proj",
			"file_tree":      "src/\n  main.go\n  config.go\n  unrelated.go",
			"open_files": []interface{}{
				map[string]interface{}{
					"path": "src/main.go", "language": "go", "content": "package main\n", "is_active": true,
				},
				map[string]interface{}{
					"path": "src/config.go", "language": "go", "content": "package config\n", "is_active": false,
				},
				map[string]interface{}{
					"path": "src/unrelated.go", "language": "go", "content": "const unrelated = true\n", "is_active": false,
				},
			},
		},
	}

	if got := ResolveContextScopeForChannel(msg, protocol.ChannelTypeDM); got != ContextScopeFocus {
		t.Fatalf("effective scope = %q, want focus", got)
	}
	var b strings.Builder
	AppendWorkspaceContextForChannel(&b, msg, protocol.ChannelTypeDM)
	out := b.String()
	if !strings.Contains(out, "Open files (2)") {
		t.Fatalf("expected physically trimmed open-file count, got:\n%s", out)
	}
	for _, want := range []string{"### src/main.go", "### src/config.go"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q after focus trimming, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "### src/unrelated.go") || strings.Contains(out, "const unrelated") {
		t.Fatalf("unrelated full-scope file leaked into focused prompt:\n%s", out)
	}
}

func TestAppendWorkspaceContext_ScanSummary(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-biologyexpert", protocol.AgentInfo{Name: "u"}, "review the scan")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeFocus,
		"workspace_context": map[string]interface{}{
			"workspace_name": "scan",
			"workspace_path": "/scan",
			"scan_summary": map[string]interface{}{
				"summary_dir": "run-summary",
				"wells_count": float64(96),
				"analytes":    []interface{}{"IL-6", "TNF-alpha"},
				"note":        "metadata only; no pixels",
				"active_well": map[string]interface{}{
					"well":       "A1",
					"spot_count": float64(2),
					"spots": []interface{}{
						map[string]interface{}{"analyte": "IL-6", "row": "1", "column": "1", "x_px": float64(49), "y_px": float64(50)},
					},
				},
			},
		},
	}
	var b strings.Builder
	AppendWorkspaceContext(&b, msg)
	out := b.String()
	for _, want := range []string{"Phoenix scan summary context", "Wells with metadata: 96", "IL-6, TNF-alpha", "Active well: A1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}
