package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// RepairFailureKind categorizes implementation-session repair feedback (ComPilot-style typed feedback).
type RepairFailureKind string

const (
	RepairFailurePreflight      RepairFailureKind = "preflight"
	RepairFailureBuild          RepairFailureKind = "build"
	RepairFailureTest           RepairFailureKind = "test"
	RepairFailurePolicy         RepairFailureKind = "policy"
	RepairFailurePartialSuccess RepairFailureKind = "success_partial"
	RepairFailureAdvisory       RepairFailureKind = "advisory"
	RepairFailureGrounding      RepairFailureKind = "grounding"
	RepairFailureVerify         RepairFailureKind = "verify"
)

var (
	tsErrorLineRE   = regexp.MustCompile(`(?m)(?:error TS\d+|SyntaxError|Module not found|Cannot find module|failed to compile)`)
	testFailLineRE  = regexp.MustCompile(`(?m)(?:FAIL\s|AssertionError|Test Suites:.*failed|tests? failed|panic:|--- FAIL:)`)
	buildFailLineRE = regexp.MustCompile(`(?m)(?:error:|Error:|ELIFECYCLE|exit_code=[1-9]|command failed|build failed)`)
)

// VerifyFailureInfo captures structured verify failure metadata for repair notes and outcomes.
type VerifyFailureInfo struct {
	Kind          RepairFailureKind
	FailedCommand string
	Summary       string
	BuildPassed   bool
}

func classifyVerifyFailure(output string, cmds []string) VerifyFailureInfo {
	info := VerifyFailureInfo{Kind: RepairFailureVerify, Summary: "verification failed"}
	output = strings.TrimSpace(output)
	if output == "" {
		return info
	}

	sections := strings.Split(output, "\n---\n")
	buildPassed := false
	for i, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		cmd := ""
		if i < len(cmds) {
			cmd = cmds[i]
		} else {
			cmd = extractVerifyCommandFromSection(section)
		}
		failed := strings.Contains(section, "exit_code=") && !strings.Contains(section, "exit_code=0")
		if !failed && (buildFailLineRE.MatchString(section) || tsErrorLineRE.MatchString(section)) {
			failed = true
		}
		if !failed {
			if isBuildVerifyCommand(cmd) {
				buildPassed = true
			}
			continue
		}
		info.FailedCommand = cmd
		if testFailLineRE.MatchString(section) || isTestVerifyCommand(cmd) {
			if buildPassed || isTestVerifyCommand(cmd) {
				info.Kind = RepairFailureTest
				if buildPassed && !isTestVerifyCommand(cmd) {
					info.Kind = RepairFailurePartialSuccess
				}
			} else {
				info.Kind = RepairFailureBuild
			}
		} else if tsErrorLineRE.MatchString(section) || isBuildVerifyCommand(cmd) {
			info.Kind = RepairFailureBuild
		} else {
			info.Kind = RepairFailureVerify
		}
		info.Summary = summarizeVerifySection(section)
		return info
	}
	info.Summary = summarizeVerifySection(output)
	return info
}

func extractVerifyCommandFromSection(section string) string {
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "$ ") {
			return strings.TrimPrefix(line, "$ ")
		}
	}
	return ""
}

func isBuildVerifyCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	return strings.Contains(lower, " build") ||
		strings.Contains(lower, "tsc") ||
		strings.Contains(lower, "compile") ||
		strings.Contains(lower, "cargo build") ||
		strings.Contains(lower, "terraform validate")
}

func isTestVerifyCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	return strings.Contains(lower, " test") ||
		strings.Contains(lower, "pytest") ||
		strings.Contains(lower, "jest")
}

func summarizeVerifySection(section string) string {
	lines := strings.Split(section, "\n")
	var picks []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "$ ") {
			continue
		}
		if tsErrorLineRE.MatchString(line) || testFailLineRE.MatchString(line) || buildFailLineRE.MatchString(line) {
			picks = append(picks, line)
			if len(picks) >= 3 {
				break
			}
		}
	}
	if len(picks) == 0 {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "$ ") {
				picks = append(picks, line)
				if len(picks) >= 2 {
					break
				}
			}
		}
	}
	if len(picks) == 0 {
		return "see command output below"
	}
	return strings.Join(picks, "; ")
}

func formatTypedRepairNote(kind RepairFailureKind, headline string, detail string, rawOutput string) string {
	var b strings.Builder
	b.WriteString("Feedback category: ")
	b.WriteString(string(kind))
	b.WriteString("\n")
	if headline != "" {
		b.WriteString(headline)
		b.WriteString("\n")
	}
	if detail != "" {
		b.WriteString("Details: ")
		b.WriteString(detail)
		b.WriteString("\n")
	}
	switch kind {
	case RepairFailurePreflight:
		b.WriteString("Fix path/stack issues, then emit corrected [FILE_CHANGE] blocks or call propose_file_edit.\n")
	case RepairFailureBuild:
		b.WriteString("Build/compile failed — fix the reported errors and re-propose edits.\n")
	case RepairFailureTest:
		b.WriteString("Tests failed — fix failing assertions while preserving the build.\n")
	case RepairFailurePartialSuccess:
		b.WriteString("Build passed but tests failed — focus on test failures next.\n")
	case RepairFailurePolicy:
		b.WriteString("A platform policy blocked the command — read required files or apply an edit before retrying.\n")
	case RepairFailureGrounding:
		b.WriteString("Read the stack manifest and workspace files before proposing edits.\n")
	case RepairFailureAdvisory:
		b.WriteString("Advice-only replies do not complete this session — ship concrete file changes now.\n")
	default:
		b.WriteString("Fix the issues and emit corrected [FILE_CHANGE] blocks.\n")
	}
	if raw := strings.TrimSpace(rawOutput); raw != "" {
		b.WriteString("\nCommand output:\n")
		b.WriteString(truncateImplLog(raw, 2500))
	}
	return b.String()
}

func formatVerifyRepairNote(info VerifyFailureInfo, rawOutput string) string {
	headline := "Verification failed."
	switch info.Kind {
	case RepairFailureBuild:
		headline = "Build/compile verification failed."
	case RepairFailureTest:
		headline = "Test verification failed."
	case RepairFailurePartialSuccess:
		headline = "Build succeeded but tests failed."
	}
	if cmd := strings.TrimSpace(info.FailedCommand); cmd != "" {
		headline += " Command: " + cmd + "."
	}
	return formatTypedRepairNote(info.Kind, headline, info.Summary, rawOutput)
}

func formatPolicyRepairNote(err error) string {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	return formatTypedRepairNote(RepairFailurePolicy, "Command blocked by implementation session policy.", detail, "")
}

func formatGroundingRepairNote(detail string) string {
	if detail == "" {
		detail = "Read the stack manifest and use read_file/glob_file_search on real paths before proposing edits."
	}
	return formatTypedRepairNote(RepairFailureGrounding, "Grounding required before file proposals.", detail, "")
}

func formatAdvisoryOnlyRepairNote() string {
	return formatTypedRepairNote(
		RepairFailureAdvisory,
		"Session requires file changes in this turn.",
		"Emit search_replace, apply_patch, propose_file_edit, or [FILE_CHANGE] blocks with real paths and content.",
		"",
	)
}

func formatAntiStopRepairNote(reason string) string {
	headline := "Continue exploring — the session is not complete."
	if reason != "" {
		headline = reason
	}
	return formatTypedRepairNote(
		RepairFailureAdvisory,
		headline,
		"Verification still fails or required files remain. Propose the next concrete fix instead of stopping.",
		"",
	)
}

func formatPreflightTypedRepairNote(errors []string, manifest *StackManifest) string {
	note := formatPreflightRepairNote(errors, manifest)
	return formatTypedRepairNote(RepairFailurePreflight, "Proposal preflight failed.", strings.TrimPrefix(note, "Proposal preflight failed:\n"), "")
}

func (s *ImplementationSessionState) recordRepairFailureKind(kind RepairFailureKind) {
	if s == nil {
		return
	}
	s.LastRepairFailureKind = kind
}

func repairFailureKindLabel(kind RepairFailureKind) string {
	if kind == "" {
		return ""
	}
	return string(kind)
}

func (s *ImplementationSessionState) failureTypeForOutcome() string {
	if s == nil {
		return ""
	}
	if s.LastVerifyFailureKind != "" {
		return string(s.LastVerifyFailureKind)
	}
	return repairFailureKindLabel(s.LastRepairFailureKind)
}

func outcomeScore(outcome map[string]interface{}) int {
	if outcome == nil {
		return 0
	}
	switch outcome["outcome"] {
	case "applied_and_verified":
		return 100
	case "proposals_submitted":
		if failed, _ := outcome["verify_failed"].(bool); !failed {
			return 80
		}
		return 50
	case "applied_verify_failed":
		return 40
	case "proposal_registration_failed":
		return 20
	default:
		return 0
	}
}

func formatBestOfKOutcomeNote(run, total int) string {
	return fmt.Sprintf("Best-of-%d run %d/%d selected.", total, run, total)
}
