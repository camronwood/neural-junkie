package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	diagnoseSectionRE = regexp.MustCompile(`(?is)(?:^|\n)\s*(?:analysis|diagnosis|root\s*cause|planned\s*(?:edits|changes)|hypothesis)\s*:`)
	diagnoseBulletRE  = regexp.MustCompile(`(?m)^\s*[-*]\s+\S`)
)

func requiresDiagnoseGate(msg *protocol.Message, state *ImplementationSessionState, wsPath string) bool {
	if state == nil || state.DiagnosePhaseComplete {
		return false
	}
	if state.FixLikeIntent {
		return false
	}
	if state.BootFixIntent {
		return true
	}
	if state.StackManifest == nil || wsPath == "" {
		return false
	}
	seed := msg.Content
	if userAffirmsPendingImplementation(msg.Content) {
		seed = msg.Content
	}
	remaining := remainingImplementationTargets(wsPath, state.StackManifest, seed)
	return len(remaining) > 1
}

func (s *ImplementationSessionState) diagnoseGateBlocksProposals() bool {
	if s == nil || !s.DiagnosePhaseRequired || s.DiagnosePhaseComplete {
		return false
	}
	return true
}

func responseContainsDiagnosis(response string) bool {
	text := strings.TrimSpace(response)
	if text == "" {
		return false
	}
	if diagnoseSectionRE.MatchString(text) {
		return true
	}
	if strings.Contains(strings.ToLower(text), "root cause") &&
		(strings.Contains(strings.ToLower(text), "file") || strings.Contains(strings.ToLower(text), "fix")) {
		return true
	}
	bullets := diagnoseBulletRE.FindAllString(text, -1)
	return len(bullets) >= 2 && len(text) >= 120
}

func formatDiagnoseRequiredRepairNote(state *ImplementationSessionState) string {
	var detail strings.Builder
	detail.WriteString("Before proposing edits, provide a short structured analysis:\n")
	detail.WriteString("- Analysis: stack structure and suspected root cause\n")
	detail.WriteString("- Planned edits: which files you will change and why\n")
	if state != nil && state.StackManifest != nil {
		detail.WriteString(state.StackManifest.FormatRepairHints())
	}
	return formatTypedRepairNote(
		RepairFailureGrounding,
		"Diagnose before editing.",
		detail.String(),
		"",
	)
}

func formatDiagnoseCompleteRepairNote() string {
	return formatTypedRepairNote(
		RepairFailureAdvisory,
		"Analysis received.",
		"Now ship concrete file changes with propose_file_edit, search_replace, or [FILE_CHANGE] blocks.",
		"",
	)
}
