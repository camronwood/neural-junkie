package agent

import (
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var cadToolNames = map[string]bool{
	"write_openscad":         true,
	"render_openscad":        true,
	"list_openscad_params":   true,
	"export_cad":             true,
}

func (a *Agent) resetCADWrittenPaths() {
	a.cadWrittenMu.Lock()
	a.cadWrittenPaths = nil
	a.cadWrittenMu.Unlock()
}

func (a *Agent) trackCADFileWritten(wsRoot, absPath string) {
	wsRoot = strings.TrimSpace(wsRoot)
	absPath = strings.TrimSpace(absPath)
	if wsRoot == "" || absPath == "" {
		return
	}
	rel, err := filepath.Rel(wsRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	a.cadWrittenMu.Lock()
	a.cadWrittenPaths = append(a.cadWrittenPaths, filepath.ToSlash(rel))
	a.cadWrittenMu.Unlock()
}

func (a *Agent) takeCADWrittenPaths() []string {
	a.cadWrittenMu.Lock()
	paths := a.cadWrittenPaths
	a.cadWrittenPaths = nil
	a.cadWrittenMu.Unlock()
	return paths
}

func (a *Agent) rewriteCADToolInput(msg *protocol.Message, name string, input json.RawMessage) json.RawMessage {
	if !cadToolNames[name] {
		return input
	}
	wsRoot := a.resolveWorkspacePath(msg)
	if wsRoot == "" {
		return input
	}

	var args map[string]interface{}
	if len(input) > 0 {
		_ = json.Unmarshal(input, &args)
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	if name == "write_openscad" {
		path, _ := args["path"].(string)
		projectID, _ := args["project_id"].(string)
		if strings.TrimSpace(path) == "" && strings.TrimSpace(projectID) == "" {
			if p := cadScadPathFromMessage(msg, wsRoot); p != "" {
				args["path"] = p
			}
		}
	}

	resolve := func(key string) {
		raw, _ := args[key].(string)
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		resolved, err := resolveCADPathWithinWorkspace(wsRoot, raw)
		if err != nil {
			log.Printf("[CAD] workspace path resolve failed for %q (%s): %v", raw, key, err)
			return
		}
		args[key] = resolved
	}

	switch name {
	case "write_openscad", "render_openscad", "list_openscad_params":
		resolve("path")
		if name == "render_openscad" {
			resolve("output_path")
		}
	case "export_cad":
		resolve("dest_dir")
		resolve("source_scad")
		resolve("source_stl")
	}

	out, err := json.Marshal(args)
	if err != nil {
		return input
	}
	return out
}

func resolveCADPathWithinWorkspace(wsRoot, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	candidate := raw
	if !filepath.IsAbs(raw) {
		candidate = filepath.Join(wsRoot, raw)
	}
	return pathutil.WithinRoot(wsRoot, candidate)
}

func cadScadPathFromMessage(msg *protocol.Message, wsRoot string) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	cadRaw, ok := ctxMap["cad"]
	if !ok || cadRaw == nil {
		return ""
	}
	cad, ok := cadRaw.(map[string]interface{})
	if !ok {
		return ""
	}
	path, _ := cad["scad_path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	resolved, err := resolveCADPathWithinWorkspace(wsRoot, path)
	if err != nil {
		return joinWorkspacePath(wsRoot, path)
	}
	return resolved
}

func cadWrittenPathFromToolInput(wsRoot string, input json.RawMessage) string {
	if wsRoot == "" || len(input) == 0 {
		return ""
	}
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return ""
	}
	resolved, err := resolveCADPathWithinWorkspace(wsRoot, args.Path)
	if err != nil {
		return ""
	}
	return resolved
}
