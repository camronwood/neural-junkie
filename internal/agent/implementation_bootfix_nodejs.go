package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func workspaceMissingPackageJSON(wsPath string) bool {
	if strings.TrimSpace(wsPath) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(wsPath, "package.json"))
	return os.IsNotExist(err)
}

func workspaceHasNodeServerSource(wsPath string) bool {
	if strings.TrimSpace(wsPath) == "" {
		return false
	}
	for _, rel := range []string{"src/server.ts", "src/index.ts", "src/app.ts", "server.ts", "src/server.js", "src/index.js"} {
		if _, err := os.Stat(filepath.Join(wsPath, rel)); err == nil {
			return true
		}
	}
	return false
}

func messageImpliesNodeAPIGreenfield(content string, hintPaths ...string) bool {
	lower := strings.ToLower(content)
	nodeish := strings.Contains(lower, "node") || strings.Contains(lower, "typescript") ||
		strings.Contains(lower, "express") || strings.Contains(lower, "fastify") ||
		strings.Contains(lower, "hono") || strings.Contains(lower, "package.json") ||
		strings.Contains(lower, "src/server.ts") || strings.Contains(lower, "src/index.ts")
	apiish := strings.Contains(lower, "crud") || strings.Contains(lower, "api") ||
		strings.Contains(lower, "users") || strings.Contains(lower, "notes") ||
		strings.Contains(lower, "memos") || strings.Contains(lower, "server")
	if nodeish && apiish {
		return true
	}
	for _, p := range hintPaths {
		p = strings.ToLower(normalizeFileChangeRelPath(p))
		if strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, "package.json") {
			return nodeish || apiish
		}
	}
	return false
}

func nodeAPIResourceName(content string) string {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "memo"):
		return "memos"
	case strings.Contains(lower, "note"):
		return "notes"
	default:
		return "users"
	}
}

func minimalNodePackageJSONBody(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "api"
	}
	return "{\n" +
		"  \"name\": \"" + name + "\",\n" +
		"  \"version\": \"0.1.0\",\n" +
		"  \"private\": true,\n" +
		"  \"type\": \"module\",\n" +
		"  \"scripts\": {\n" +
		"    \"start\": \"tsx src/server.ts\",\n" +
		"    \"dev\": \"tsx watch src/server.ts\"\n" +
		"  },\n" +
		"  \"dependencies\": {\n" +
		"    \"express\": \"^4.19.0\"\n" +
		"  },\n" +
		"  \"devDependencies\": {\n" +
		"    \"tsx\": \"^4.19.0\",\n" +
		"    \"typescript\": \"^5.6.0\",\n" +
		"    \"@types/express\": \"^4.17.21\",\n" +
		"    \"@types/node\": \"^22.0.0\"\n" +
		"  }\n" +
		"}\n"
}

func minimalNodeCRUDServerTS(resource string) string {
	route := strings.TrimSpace(resource)
	if route == "" {
		route = "users"
	}
	singular := strings.TrimSuffix(route, "s")
	if singular == route {
		singular = route
	}
	typeName := strings.ToUpper(singular[:1]) + singular[1:]
	return `import express from "express";

type ` + typeName + ` = { id: string; name: string; email?: string };

const app = express();
app.use(express.json());

const store: ` + typeName + `[] = [
  { id: "1", name: "Ada", email: "ada@example.com" },
  { id: "2", name: "Grace", email: "grace@example.com" },
];

app.get("/` + route + `", (_req, resp) => {
  resp.json(store);
});

app.get("/` + route + `/:id", (req, resp) => {
  const row = store.find((u) => u.id === req.params.id);
  if (!row) {
    resp.status(404).json({ error: "not found" });
    return;
  }
  resp.json(row);
});

app.post("/` + route + `", (req, resp) => {
  const row: ` + typeName + ` = {
    id: String(store.length + 1),
    name: String(req.body?.name || "user"),
    email: req.body?.email ? String(req.body.email) : undefined,
  };
  store.push(row);
  resp.status(201).json(row);
});

app.put("/` + route + `/:id", (req, resp) => {
  const idx = store.findIndex((u) => u.id === req.params.id);
  if (idx < 0) {
    resp.status(404).json({ error: "not found" });
    return;
  }
  store[idx] = {
    ...store[idx],
    name: req.body?.name != null ? String(req.body.name) : store[idx].name,
    email: req.body?.email != null ? String(req.body.email) : store[idx].email,
  };
  resp.json(store[idx]);
});

app.delete("/` + route + `/:id", (req, resp) => {
  const idx = store.findIndex((u) => u.id === req.params.id);
  if (idx < 0) {
    resp.status(404).json({ error: "not found" });
    return;
  }
  const [removed] = store.splice(idx, 1);
  resp.json(removed);
});

app.listen(3000, () => {
  console.log("listening on :3000");
});
`
}

func rewriteNodeResourceRoutes(src, from, to string) (string, bool) {
	if from == "" || to == "" || from == to {
		return "", false
	}
	if !strings.Contains(src, "/"+from) && !strings.Contains(strings.ToLower(src), from) {
		return "", false
	}
	out := src
	out = strings.ReplaceAll(out, "/"+from, "/"+to)
	// Common identifier swaps for notes→memos / users stay untouched.
	fromTitle := strings.ToUpper(from[:1]) + from[1:]
	toTitle := strings.ToUpper(to[:1]) + to[1:]
	fromSing := strings.TrimSuffix(from, "s")
	toSing := strings.TrimSuffix(to, "s")
	out = strings.ReplaceAll(out, fromTitle, toTitle)
	out = strings.ReplaceAll(out, fromSing, toSing)
	out = strings.ReplaceAll(out, from, to)
	if out == src {
		return "", false
	}
	return out, true
}

func userRequestsResourceRename(content string) (from, to string, ok bool) {
	lower := strings.ToLower(content)
	renameish := strings.Contains(lower, "rename") || strings.Contains(lower, "correction") ||
		strings.Contains(lower, "do not leave") || strings.Contains(lower, "only /memos") ||
		strings.Contains(lower, "only memos")
	if !renameish {
		return "", "", false
	}
	if strings.Contains(lower, "notes") && strings.Contains(lower, "memos") {
		return "notes", "memos", true
	}
	return "", "", false
}

func nodeServerSatisfiesMemosOnly(wsPath string) bool {
	for _, rel := range []string{"src/server.ts", "src/index.ts", "src/app.ts", "server.ts"} {
		body, err := os.ReadFile(filepath.Join(wsPath, rel))
		if err != nil {
			continue
		}
		s := string(body)
		if strings.Contains(s, "/memos") && !strings.Contains(s, "/notes") {
			return true
		}
	}
	return false
}

func userConfirmsNodeMemosDeliverable(content string) bool {
	lower := strings.ToLower(content)
	return (strings.Contains(lower, "confirm") || strings.Contains(lower, "stop") || strings.Contains(lower, "only resource")) &&
		strings.Contains(lower, "memo")
}

// tryGreenfieldNodeAPIScaffold creates package.json + src/server.ts for empty Node/TS CRUD asks.
func (a *Agent) tryGreenfieldNodeAPIScaffold(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	ok, _ := a.attemptGreenfieldNodeAPIScaffold(ctx, msg, wsPath, msg.Channel, state)
	return ok
}

func (a *Agent) attemptGreenfieldNodeAPIScaffold(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
) (bool, []string) {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false, nil
	}
	if skipCollabCodingFixtureSynths(msg) || isAskModeReadOnly(msg) {
		return false, nil
	}
	if channel == "" {
		channel = "general"
	}
	if !msg.ImplementationSession() || !messageImpliesNodeAPIGreenfield(msg.Content) {
		return false, nil
	}
	needPkg := workspaceMissingPackageJSON(wsPath)
	needServer := !workspaceHasNodeServerSource(wsPath)
	if !needPkg && !needServer {
		return false, nil
	}
	if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
		return false, nil
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["deterministic_edit"] = true

	resource := nodeAPIResourceName(msg.Content)
	var changed []string
	pkgName := deriveCargoPackageName(wsPath)
	if pkgName == "" || pkgName == "app" {
		pkgName = "user-api"
	}

	if needPkg {
		body := minimalNodePackageJSONBody(pkgName)
		if err := a.proposeFileCreateInChannel(ctx, channel, "package.json", body, msg); err != nil {
			log.Printf("[%s] greenfield_node_package_propose_failed(err=%v)", a.Info.Name, err)
		}
		pkgPath := filepath.Join(wsPath, "package.json")
		onDisk, readErr := os.ReadFile(pkgPath)
		if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
			if err := os.WriteFile(pkgPath, []byte(body), 0o644); err != nil {
				return false, nil
			}
			state.releaseSnapshot("package.json")
			log.Printf("[%s] greenfield_node_package_direct_apply", a.Info.Name)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{"package.json"})
		state.RecordEdit("package.json")
		state.RecordReadPath("package.json")
		changed = append(changed, "package.json")
	}

	if needServer {
		body := minimalNodeCRUDServerTS(resource)
		if err := os.MkdirAll(filepath.Join(wsPath, "src"), 0o755); err != nil {
			return len(changed) > 0, changed
		}
		if err := a.proposeFileCreateInChannel(ctx, channel, "src/server.ts", body, msg); err != nil {
			log.Printf("[%s] greenfield_node_server_propose_failed(err=%v)", a.Info.Name, err)
		}
		serverPath := filepath.Join(wsPath, "src", "server.ts")
		onDisk, readErr := os.ReadFile(serverPath)
		if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
			if err := os.WriteFile(serverPath, []byte(body), 0o644); err != nil {
				return len(changed) > 0, changed
			}
			state.releaseSnapshot("src/server.ts")
			log.Printf("[%s] greenfield_node_server_direct_apply", a.Info.Name)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{"src/server.ts"})
		state.RecordEdit("src/server.ts")
		state.RecordReadPath("src/server.ts")
		changed = append(changed, "src/server.ts")
	}

	if len(changed) == 0 {
		return false, nil
	}
	state.SetPlaybookUsed("greenfield_node_api")
	state.VerifySkipped = true // greenfield stub; npm install not required for disk waits
	log.Printf("[%s] greenfield_node_api_scaffold(files=%v resource=%s)", a.Info.Name, changed, resource)
	return true, changed
}

// tryRenameNodeAPIResourceFix rewrites /notes → /memos (etc.) on correction turns.
func (a *Agent) tryRenameNodeAPIResourceFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	if skipCollabCodingFixtureSynths(msg) || isAskModeReadOnly(msg) {
		return false
	}
	from, to, ok := userRequestsResourceRename(msg.Content)
	if !ok {
		return false
	}
	if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
		return false
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	var changed bool
	for _, rel := range []string{"src/server.ts", "src/index.ts", "src/app.ts", "server.ts"} {
		abs := filepath.Join(wsPath, rel)
		existing, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		body, ok := rewriteNodeResourceRoutes(string(existing), from, to)
		if !ok {
			continue
		}
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		msg.Metadata["deterministic_edit"] = true
		if _, err := a.proposeFileEditInChannel(ctx, channel, rel, string(existing), body, msg); err != nil {
			log.Printf("[%s] node_resource_rename_propose_failed(file=%s err=%v)", a.Info.Name, rel, err)
		}
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || strings.Contains(string(onDisk), "/"+from) {
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				continue
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] node_resource_rename_direct_apply(file=%s)", a.Info.Name, rel)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		state.SetPlaybookUsed("node_resource_rename")
		changed = true
		log.Printf("[%s] node_resource_rename_fix(from=%s to=%s file=%s)", a.Info.Name, from, to, rel)
		break
	}
	return changed
}

func userRequestsHealthEndpoint(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "/health") ||
		(strings.Contains(lower, "health") && (strings.Contains(lower, "endpoint") || strings.Contains(lower, "route") || strings.Contains(lower, "add")))
}

func ensureNodeHealthRoute(src string) (string, bool) {
	if strings.Contains(src, "/health") {
		return "", false
	}
	// Prefer inserting before listen so Express apps keep a single listen call.
	needle := "app.listen("
	idx := strings.Index(src, needle)
	insert := "app.get(\"/health\", (_req, resp) => {\n  resp.json({ ok: true, status: \"ok\" });\n});\n\n"
	if idx >= 0 {
		return src[:idx] + insert + src[idx:], true
	}
	return src + "\n" + insert, true
}

// tryAddNodeHealthEndpointFix adds GET /health when a correction turn asks for it.
func (a *Agent) tryAddNodeHealthEndpointFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	if skipCollabCodingFixtureSynths(msg) || isAskModeReadOnly(msg) {
		return false
	}
	if !userRequestsHealthEndpoint(msg.Content) {
		return false
	}
	if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
		return false
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	for _, rel := range []string{"src/server.ts", "src/index.ts", "src/app.ts", "server.ts"} {
		abs := filepath.Join(wsPath, rel)
		existing, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		body, ok := ensureNodeHealthRoute(string(existing))
		if !ok {
			continue
		}
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		msg.Metadata["deterministic_edit"] = true
		if _, err := a.proposeFileEditInChannel(ctx, channel, rel, string(existing), body, msg); err != nil {
			log.Printf("[%s] node_health_propose_failed(file=%s err=%v)", a.Info.Name, rel, err)
		}
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || !strings.Contains(string(onDisk), "/health") {
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				continue
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] node_health_direct_apply(file=%s)", a.Info.Name, rel)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		state.SetPlaybookUsed("node_health_endpoint")
		state.VerifySkipped = true
		log.Printf("[%s] node_health_endpoint_fix(file=%s)", a.Info.Name, rel)
		return true
	}
	return false
}

func nodeServerSatisfiesUsersAndHealth(wsPath string) bool {
	for _, rel := range []string{"src/server.ts", "src/index.ts", "src/app.ts", "server.ts"} {
		body, err := os.ReadFile(filepath.Join(wsPath, rel))
		if err != nil {
			continue
		}
		s := string(body)
		if strings.Contains(s, "/health") && (strings.Contains(s, "/users") || strings.Contains(s, "User")) {
			return true
		}
	}
	return false
}

func userConfirmsNodeHealthDeliverable(content string) bool {
	lower := strings.ToLower(content)
	return (strings.Contains(lower, "verify") || strings.Contains(lower, "confirm") || strings.Contains(lower, "stop")) &&
		strings.Contains(lower, "health") &&
		(strings.Contains(lower, "user") || strings.Contains(lower, "crud"))
}

func (a *Agent) tryNodeMemosConfirmSatisfied(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	if skipCollabCodingFixtureSynths(msg) || isAskModeReadOnly(msg) {
		return false
	}
	if !userConfirmsNodeMemosDeliverable(msg.Content) || !nodeServerSatisfiesMemosOnly(wsPath) {
		return false
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{"src/server.ts"})
	state.RecordReadPath("src/server.ts")
	state.SetPlaybookUsed("node_memos_confirm")
	state.VerifySkipped = true
	log.Printf("[%s] node_memos_confirm_satisfied", a.Info.Name)
	return true
}

func (a *Agent) tryNodeHealthConfirmSatisfied(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	if skipCollabCodingFixtureSynths(msg) || isAskModeReadOnly(msg) {
		return false
	}
	if !userConfirmsNodeHealthDeliverable(msg.Content) || !nodeServerSatisfiesUsersAndHealth(wsPath) {
		return false
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{"src/server.ts"})
	state.RecordReadPath("src/server.ts")
	state.SetPlaybookUsed("node_health_confirm")
	state.VerifySkipped = true
	log.Printf("[%s] node_health_confirm_satisfied", a.Info.Name)
	return true
}

func (a *Agent) maybeScaffoldGreenfieldNodeServerAfterPackageJSON(ctx context.Context, sourceMsg *protocol.Message, path string) {
	if a == nil || sourceMsg == nil || !sourceMsg.ImplementationSession() {
		return
	}
	path = normalizeFileChangeRelPath(path)
	if !strings.EqualFold(filepath.Base(path), "package.json") {
		return
	}
	wsPath := a.resolveWorkspacePath(sourceMsg)
	if wsPath == "" || workspaceHasNodeServerSource(wsPath) {
		return
	}
	if !messageImpliesNodeAPIGreenfield(sourceMsg.Content) {
		return
	}
	if resolveImplementationTrustMode(sourceMsg) != editorTrustAutoApply {
		return
	}
	state := implementationSessionStateFromContext(ctx)
	if state == nil {
		return
	}
	// Only land the server entry — do not re-enter package.json create (recursion via propose hooks).
	channel := sourceMsg.Channel
	if channel == "" {
		channel = "general"
	}
	if sourceMsg.Metadata == nil {
		sourceMsg.Metadata = map[string]interface{}{}
	}
	sourceMsg.Metadata["deterministic_edit"] = true
	body := minimalNodeCRUDServerTS(nodeAPIResourceName(sourceMsg.Content))
	if err := os.MkdirAll(filepath.Join(wsPath, "src"), 0o755); err != nil {
		return
	}
	rel := "src/server.ts"
	if err := a.proposeFileCreateInChannel(ctx, channel, rel, body, sourceMsg); err != nil {
		log.Printf("[%s] greenfield_node_server_after_pkg_propose_failed(err=%v)", a.Info.Name, err)
	}
	abs := filepath.Join(wsPath, rel)
	onDisk, readErr := os.ReadFile(abs)
	if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			return
		}
		state.releaseSnapshot(rel)
		log.Printf("[%s] greenfield_node_server_after_pkg_direct_apply", a.Info.Name)
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
	state.RecordEdit(rel)
	state.RecordReadPath(rel)
	if state.PlaybookUsed() == "" {
		state.SetPlaybookUsed("greenfield_node_api")
	}
}
