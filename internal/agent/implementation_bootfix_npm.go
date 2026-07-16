package agent

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var missingNpmModuleRE = regexp.MustCompile(`(?i)(?:cannot find module|can't resolve|module not found|failed to resolve import) ['"]([^'"]+)['"]`)
var viteMissingDepRE = regexp.MustCompile(`(?m)^\s*([@a-z0-9][a-z0-9._/@-]*)\s+\(imported by`)

func extractMissingNpmModules(output string) []string {
	seen := make(map[string]bool)
	var mods []string
	add := func(mod string) {
		mod = strings.TrimSpace(mod)
		if mod == "" || strings.HasPrefix(mod, ".") || strings.HasPrefix(mod, "/") {
			return
		}
		if i := strings.IndexByte(mod, '/'); i > 0 && !strings.HasPrefix(mod, "@") {
			mod = mod[:i]
		}
		if seen[mod] {
			return
		}
		seen[mod] = true
		mods = append(mods, mod)
	}
	for _, m := range missingNpmModuleRE.FindAllStringSubmatch(output, -1) {
		if len(m) >= 2 {
			add(m[1])
		}
	}
	for _, m := range viteMissingDepRE.FindAllStringSubmatch(output, -1) {
		if len(m) >= 2 {
			add(m[1])
		}
	}
	return mods
}

func defaultNpmDependencyVersion(module string) string {
	switch module {
	case "react-bootstrap":
		return "^2.10.0"
	case "bootstrap":
		return "^5.3.0"
	default:
		return "^1.0.0"
	}
}

func addMissingDependencyToPackageJSON(existing []byte, module string) (string, bool) {
	module = strings.TrimSpace(module)
	if module == "" {
		return "", false
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal(existing, &pkg); err != nil {
		return "", false
	}
	if pkgHasDependency(pkg, module) {
		return "", false
	}
	deps, _ := pkg["dependencies"].(map[string]interface{})
	if deps == nil {
		deps = make(map[string]interface{})
		pkg["dependencies"] = deps
	}
	deps[module] = defaultNpmDependencyVersion(module)
	out, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return "", false
	}
	return string(out) + "\n", true
}

func pkgHasDependency(pkg map[string]interface{}, module string) bool {
	for _, key := range []string{"dependencies", "devDependencies", "peerDependencies", "optionalDependencies"} {
		deps, _ := pkg[key].(map[string]interface{})
		if deps == nil {
			continue
		}
		if _, ok := deps[module]; ok {
			return true
		}
	}
	return false
}

// tryEarlyMissingNpmModuleFix proposes package.json when needed and marks playbook for npm install.
func (a *Agent) tryEarlyMissingNpmModuleFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	if skipCollabCodingFixtureSynths(msg) {
		return false
	}
	evidence := strings.TrimSpace(state.LastCommandOutput())
	if evidence == "" {
		evidence = msg.Content
	}
	if commandOutputMatchesPlaybook(evidence) != "missing_npm_module" {
		return false
	}
	modules := extractMissingNpmModules(evidence)
	if len(modules) == 0 {
		return false
	}
	module := modules[0]
	if npmModuleInstalled(wsPath, module) {
		return false
	}
	pkgPath := filepath.Join(wsPath, "package.json")
	existing, err := os.ReadFile(pkgPath)
	if err != nil {
		return false
	}
	var pkg map[string]interface{}
	if json.Unmarshal(existing, &pkg) != nil {
		return false
	}
	if !pkgHasDependency(pkg, module) {
		body, ok := addMissingDependencyToPackageJSON(existing, module)
		if !ok {
			return false
		}
		if err := a.validateProposalForSession(ctx, msg, "package.json", ProposalOpEdit); err != nil {
			return false
		}
		if err := a.proposeFileEditInChannel(ctx, msg.Channel, "package.json", string(existing), body, msg); err != nil {
			return false
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{"package.json"})
		state.RecordEdit("package.json")
	}
	state.DiagnosePhaseComplete = true
	state.SetPlaybookUsed("missing_npm_module")
	log.Printf("[%s] early_missing_npm_module_fix(module=%s)", a.Info.Name, module)
	return true
}

func (a *Agent) runBootFixNpmInstallAfterDepProposal(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState, wsPath string) {
	if a == nil || msg == nil || state == nil || wsPath == "" || state.PlaybookUsed() != "missing_npm_module" {
		return
	}
	evidence := strings.TrimSpace(state.LastCommandOutput())
	if evidence == "" {
		evidence = msg.Content
	}
	modules := extractMissingNpmModules(evidence)
	if len(modules) == 0 {
		return
	}
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return
	}
	module := modules[0]
	cmd := "npm install " + module
	a.sendThinkingActivity(msg, protocol.ThinkingActivityImplementation, "Installing "+module+"…")
	toolCtx := a.contextWithWorkspaceBackend(ctx, msg)
	toolCtx = shared.ContextWithImplementationSession(toolCtx, true)
	toolCtx = shared.ContextWithRunCommandTimeout(toolCtx, 3*time.Minute)
	toolCtx = attachImplSessionCommandPolicy(toolCtx, state)
	input, _ := json.Marshal(map[string]string{"command": cmd})
	result, err := executeMCPTool(toolCtx, mcpServer, "run_command", input)
	if err != nil {
		log.Printf("[%s] fix npm install failed: %v", a.Info.Name, err)
		return
	}
	exitCode, _, _ := parseRunCommandMCPResult(result)
	state.RecordCommandRun(cmd, exitCode, result)
	if !shouldSkipDuplicateCommandBroadcast(msg.Channel, a.Info.ID, cmd, result) {
		a.broadcastAgentRunCommandOutput(msg, cmd, result)
	}
}

// completeBootFixDiagnoseFromBootstrap marks the diagnose gate satisfied when bootstrap
// command output already identifies a concrete failure class.
func completeBootFixDiagnoseFromBootstrap(state *ImplementationSessionState) {
	if state == nil || !state.BootFixIntent {
		return
	}
	if sig := commandOutputMatchesPlaybook(state.LastCommandOutput()); sig != "" {
		state.DiagnosePhaseComplete = true
	}
}
