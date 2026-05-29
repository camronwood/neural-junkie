package collaboration

import (
	"regexp"
	"strings"
)

var stalePlanningApproveRE = regexp.MustCompile(`(?im)(/approve-plan|approve the plan|use the command.*approve)`)

// AgentReplyContainsStalePlanning reports execution-phase replies that reopen plan approval.
func AgentReplyContainsStalePlanning(content string) bool {
	return stalePlanningApproveRE.MatchString(strings.TrimSpace(content))
}

// SanitizeCollabExecutionResponse strips planning-phase boilerplate from execution replies.
func SanitizeCollabExecutionResponse(content string, phase string) string {
	if strings.TrimSpace(phase) != string(PhaseExecuting) {
		return content
	}
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if stalePlanningApproveRE.MatchString(line) {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
