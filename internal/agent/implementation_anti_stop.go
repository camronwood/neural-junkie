package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	prematureDoneRE = regexp.MustCompile(`(?i)\b(?:i think (?:this |it )?is fixed|no further (?:changes|edits|transformations)|nothing (?:else|more) to (?:do|fix)|session (?:is )?complete|finished without changes)\b`)
	advisoryOnlyRE  = regexp.MustCompile(`(?i)\b(?:you (?:could|should|might)|consider (?:adding|using)|i recommend|here(?:'s| is) (?:how|what)|let me know if)\b`)
)

func responseClaimsPrematureDone(response string) bool {
	return prematureDoneRE.MatchString(strings.TrimSpace(response))
}

func responseIsAdvisoryOnly(response string) bool {
	text := strings.TrimSpace(response)
	if text == "" {
		return true
	}
	if fileChangeBlockRegex.MatchString(text) {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "propose_file_edit") ||
		strings.Contains(lower, "search_replace") ||
		strings.Contains(lower, "apply_patch") {
		return false
	}
	if len(text) < 80 {
		return true
	}
	return advisoryOnlyRE.MatchString(text) && !strings.Contains(lower, "[file_change]")
}

func shouldRejectPrematureStop(
	a *Agent,
	msg *protocol.Message,
	state *ImplementationSessionState,
	cycleProposed bool,
	round int,
	maxRounds int,
) (bool, string) {
	if state == nil || msg == nil || cycleProposed {
		return false, ""
	}
	if round >= maxRounds-1 {
		return false, ""
	}

	wsPath := ""
	if a != nil {
		wsPath = a.resolveWorkspacePath(msg)
	}
	remaining := remainingImplementationTargets(wsPath, state.StackManifest, msg.Content)
	if len(remaining) > 0 {
		return true, formatAntiStopRepairNote("Required files remain: " + strings.Join(remaining, ", "))
	}
	if state.VerifyFailed {
		return true, formatAntiStopRepairNote("Verification still fails.")
	}
	if state.PrematureStopAttempts >= 5 {
		return false, ""
	}
	return false, ""
}

func notePrematureStopAttempt(state *ImplementationSessionState) {
	if state == nil {
		return
	}
	state.PrematureStopAttempts++
}
