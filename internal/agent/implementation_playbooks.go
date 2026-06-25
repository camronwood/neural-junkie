package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// tryEarlyMissingStartAllMakefileFix wires the Makefile start-all playbook before LLM rounds.
func (a *Agent) tryEarlyMissingStartAllMakefileFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	sig := commandOutputMatchesPlaybook(msg.Content)
	if sig == "" {
		sig = "missing_start_all_target"
		if !strings.Contains(strings.ToLower(msg.Content), "start-all") {
			return false
		}
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
		sig = commandOutputMatchesPlaybook(msg.Content)
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
		if err := a.proposeFileCreateInChannel(channel, "Makefile", body, msg); err != nil {
			return false, nil
		}
	} else {
		if err := a.proposeFileEditInChannel(channel, "Makefile", oldContent, body, msg); err != nil {
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
	_ = ctx
	_ = msg
	_ = channel
	_ = state
	tauriConf := filepath.Join(wsPath, "src-tauri", "tauri.conf.json")
	viteConfigs := []string{"vite.config.ts", "vite.config.js", "vite.config.mjs"}
	if _, err := os.Stat(tauriConf); err != nil {
		return false, nil
	}
	var vitePath string
	for _, p := range viteConfigs {
		if _, err := os.Stat(filepath.Join(wsPath, p)); err == nil {
			vitePath = p
			break
		}
	}
	if vitePath == "" {
		return false, nil
	}
	// Port alignment is stack-specific; leave to model after reads — playbook marks intent only.
	if state != nil {
		state.SetPlaybookUsed("tauri_vite_port_mismatch")
	}
	return false, nil
}
