package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// tryEarlyCommandEvidencePlaybook applies a parity playbook only when the user message
// already contains pasted command failure output (scenario tests, error-log paste).
func (a *Agent) tryEarlyCommandEvidencePlaybook(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	sig := playbookSignatureFromCommandEvidence(msg.Content)
	if sig == "" {
		return false
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	ok, _ := a.attemptCommandFailurePlaybook(ctx, msg, wsPath, channel, sig, state)
	return ok
}

func (a *Agent) attemptCommandFailurePlaybook(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel, signature string,
	state *ImplementationSessionState,
) (bool, []string) {
	if a == nil || msg == nil || wsPath == "" {
		return false, nil
	}
	switch signature {
	case "missing_start_all_target":
		return a.attemptMissingStartAllMakefileFix(ctx, msg, wsPath, channel, state)
	case "tauri_vite_port_mismatch":
		return a.attemptTauriVitePortPlaybook(ctx, msg, wsPath, channel, state)
	default:
		return false, nil
	}
}

func (a *Agent) attemptPlaybookForSessionState(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) (bool, []string) {
	if state == nil {
		return false, nil
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return false, nil
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	sig := commandOutputMatchesPlaybook(state.LastCommandOutput())
	if sig == "" {
		sig = playbookSignatureFromCommandEvidence(msg.Content)
	}
	if sig == "" {
		return false, nil
	}
	return a.attemptCommandFailurePlaybook(ctx, msg, wsPath, channel, sig, state)
}

func (a *Agent) attemptMissingStartAllMakefileFix(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
) (bool, []string) {
	scriptPath := filepath.Join(wsPath, "scripts", "start-all.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		return false, nil
	}
	makefilePath := filepath.Join(wsPath, "Makefile")
	existing, err := os.ReadFile(makefilePath)
	body := ""
	if err != nil {
		body = synthesizeMakefileWithStartAll("")
	} else {
		existingStr := string(existing)
		if strings.Contains(existingStr, "start-all:") || strings.Contains(existingStr, "start-all ") {
			return false, nil
		}
		body = synthesizeMakefileWithStartAll(existingStr)
	}
	if body == "" {
		return false, nil
	}
	oldContent := string(existing)
	if err != nil {
		oldContent = ""
	}
	if err := a.validateProposalForSession(ctx, msg, "Makefile", inferProposalOp(wsPath, "Makefile", oldContent)); err != nil {
		if state.BootFixIntent && commandOutputMatchesPlaybook(state.LastCommandOutput()+msg.Content) == "missing_start_all_target" {
			if err2 := ValidateProposal(wsPath, "Makefile", inferProposalOp(wsPath, "Makefile", oldContent), a.manifestForProposal(ctx, msg)); err2 != nil {
				return false, nil
			}
		} else {
			return false, nil
		}
	}
	if oldContent == "" {
		if err := a.proposeFileCreateInChannel(ctx, msg.Channel, "Makefile", body, msg); err != nil {
			return false, nil
		}
	} else {
		if err := a.proposeFileEditInChannel(ctx, msg.Channel, "Makefile", oldContent, body, msg); err != nil {
			return false, nil
		}
	}
	if state != nil {
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{"Makefile"})
		state.RecordEdit("Makefile")
		state.SetPlaybookUsed("missing_start_all_target")
	}
	log.Printf("[%s] playbook_missing_start_all_target", a.Info.Name)
	return true, []string{"Makefile"}
}

func inferProposalOp(wsPath, path, existing string) ProposalOperation {
	if existing == "" {
		return ProposalOpCreate
	}
	return ProposalOpEdit
}

func synthesizeMakefileWithStartAll(existing string) string {
	block := strings.TrimSpace(`
.PHONY: start-all
start-all:
	@bash scripts/start-all.sh
`)
	if strings.TrimSpace(existing) == "" {
		return block + "\n"
	}
	trim := strings.TrimRight(existing, "\n")
	if strings.Contains(existing, ".PHONY") {
		if !strings.Contains(existing, "start-all") {
			trim += "\n\n" + block
		}
		return trim + "\n"
	}
	return trim + "\n\n" + block + "\n"
}

func (a *Agent) attemptTauriVitePortPlaybook(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
) (bool, []string) {
	_ = channel
	tauriConf := filepath.Join(wsPath, "src-tauri", "tauri.conf.json")
	viteConfigs := []string{"vite.config.ts", "vite.config.js", "vite.config.mjs"}
	if _, err := os.Stat(tauriConf); err != nil {
		return false, nil
	}
	var viteRel string
	for _, p := range viteConfigs {
		if _, err := os.Stat(filepath.Join(wsPath, p)); err == nil {
			viteRel = p
			break
		}
	}
	if viteRel == "" {
		return false, nil
	}
	tauriData, err := os.ReadFile(tauriConf)
	if err != nil {
		return false, nil
	}
	tauriPort := extractLocalhostPort(string(tauriData))
	if tauriPort <= 0 {
		return false, nil
	}
	vitePath := filepath.Join(wsPath, viteRel)
	viteData, err := os.ReadFile(vitePath)
	if err != nil {
		return false, nil
	}
	vitePort := extractViteServerPort(string(viteData))
	if vitePort <= 0 || vitePort == tauriPort {
		if state != nil {
			state.SetPlaybookUsed("tauri_vite_port_mismatch")
		}
		return false, nil
	}
	newBody := replaceViteServerPort(string(viteData), tauriPort)
	if newBody == string(viteData) {
		return false, nil
	}
	oldContent := string(viteData)
	if err := a.validateProposalForSession(ctx, msg, viteRel, inferProposalOp(wsPath, viteRel, oldContent)); err != nil {
		return false, nil
	}
	if err := a.proposeFileEditInChannel(ctx, msg.Channel, viteRel, oldContent, newBody, msg); err != nil {
		return false, nil
	}
	if state != nil {
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{viteRel})
		state.RecordEdit(viteRel)
		state.SetPlaybookUsed("tauri_vite_port_mismatch")
	}
	log.Printf("[%s] playbook_tauri_vite_port_mismatch vite=%d tauri=%d", a.Info.Name, vitePort, tauriPort)
	return true, []string{viteRel}
}

func extractLocalhostPort(text string) int {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, "http://localhost:")
	if idx < 0 {
		idx = strings.Index(lower, "https://localhost:")
	}
	if idx < 0 {
		return 0
	}
	rest := text[idx:]
	end := strings.IndexAny(rest, "\"' \t\n\r,}")
	if end < 0 {
		end = len(rest)
	}
	host := rest[:end]
	colon := strings.LastIndex(host, ":")
	if colon < 0 {
		return 0
	}
	portStr := strings.TrimRight(host[colon+1:], "/")
	n, err := strconv.Atoi(portStr)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func extractViteServerPort(content string) int {
	re := regexp.MustCompile(`(?i)server\s*:\s*\{[^}]*port\s*:\s*(\d+)`)
	if m := re.FindStringSubmatch(content); len(m) >= 2 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	re2 := regexp.MustCompile(`(?i)port\s*:\s*(\d+)`)
	if m := re2.FindStringSubmatch(content); len(m) >= 2 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func replaceViteServerPort(content string, port int) string {
	re := regexp.MustCompile(`(?i)(server\s*:\s*\{[^}]*port\s*:\s*)(\d+)`)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, fmt.Sprintf("${1}%d", port))
	}
	if strings.Contains(content, "defineConfig") && strings.Contains(content, "server:") {
		return content
	}
	if strings.Contains(content, "defineConfig({") {
		return strings.Replace(content, "defineConfig({", fmt.Sprintf("defineConfig({\n  server: { port: %d, strictPort: true },", port), 1)
	}
	return content
}
