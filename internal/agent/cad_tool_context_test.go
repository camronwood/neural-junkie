package agent

import (
	"encoding/json"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestRewriteCADToolInputJoinsWorkspace(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Type: protocol.AgentTypeCAD},
	}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/Users/test/cad-test",
			},
		},
	}
	out := a.rewriteCADToolInput(msg, "write_openscad", json.RawMessage(`{"path":"ball.scad","content":"sphere(10);"}`))
	var args map[string]string
	if err := json.Unmarshal(out, &args); err != nil {
		t.Fatal(err)
	}
	want := "/Users/test/cad-test/ball.scad"
	if args["path"] != want {
		t.Fatalf("path = %q, want %q", args["path"], want)
	}
}

func TestRewriteCADToolInputUsesCadContextWhenPathMissing(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeCAD}}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/Users/test/cad-test",
				"cad": map[string]interface{}{
					"scad_path": "model.scad",
				},
			},
		},
	}
	out := a.rewriteCADToolInput(msg, "write_openscad", json.RawMessage(`{"content":"cube(10);"}`))
	var args map[string]string
	if err := json.Unmarshal(out, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "/Users/test/cad-test/model.scad" {
		t.Fatalf("path = %q", args["path"])
	}
}

func TestRewriteCADToolInputPreservesAbsolutePath(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeCAD}}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/Users/test/cad-test",
			},
		},
	}
	abs := "/Users/test/cad-test/existing.scad"
	out := a.rewriteCADToolInput(msg, "render_openscad", json.RawMessage(`{"path":"`+abs+`"}`))
	var args map[string]string
	if err := json.Unmarshal(out, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != abs {
		t.Fatalf("path = %q", args["path"])
	}
}

func TestTrackCADFileWrittenRelativePath(t *testing.T) {
	a := &Agent{}
	a.trackCADFileWritten("/Users/test/cad-test", "/Users/test/cad-test/ball.scad")
	paths := a.takeCADWrittenPaths()
	if len(paths) != 1 || paths[0] != "ball.scad" {
		t.Fatalf("paths = %v", paths)
	}
}
