package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var vagueFixReportRE = regexp.MustCompile(`(?i)\b(broken|not working|doesn't work|does not work|won't work|something'?s wrong|help me fix|can you fix)\b`)

func reproTargetAmbiguous(wsPath string, manifest *StackManifest, userText string) bool {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return true
	}
	if messageHasBootOrBuildError(userText) {
		return false
	}
	if commandOutputMatchesPlaybook(userText) != "" {
		return false
	}
	if inferReproCommand(wsPath, manifest, userText) != "" {
		return false
	}
	return vagueFixReportRE.MatchString(userText)
}

// maybeAskFixClarification returns a short question when repro target cannot be inferred.
func maybeAskFixClarification(msg *protocol.Message, state *ImplementationSessionState, wsPath string) (string, bool) {
	if msg == nil || state == nil || !state.FixLikeIntent {
		return "", false
	}
	if state.ClarifyQuestionsAsked >= 2 {
		return "", false
	}
	if !reproTargetAmbiguous(wsPath, state.StackManifest, msg.Content) {
		return "", false
	}
	state.ClarifyQuestionsAsked++
	return "What command fails, or paste the error output? I'll run it and fix the root cause.", true
}

// messageImpliesFixLikeIntent reports broken/error/fix language without pure feature-build asks.
func messageImpliesFixLikeIntent(content string, history []*protocol.Message) bool {
	if !userRequestsImplementation(content) {
		return false
	}
	if isAdvisoryImplementationQuestion(content) {
		return false
	}
	if themeImplementationRE.MatchString(content) && !messageHasBootOrBuildError(content) {
		lower := strings.ToLower(content)
		if !strings.Contains(lower, "broken") && !strings.Contains(lower, "not working") &&
			!strings.Contains(lower, "fix") && !strings.Contains(lower, "error") {
			return false
		}
	}
	if messageImpliesBootFix(content, history) || messageHasBootOrBuildError(content) {
		return true
	}
	if commandOutputMatchesPlaybook(content) != "" {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(content))
	fixPhrases := []string{
		"not working", "doesn't work", "does not work", "broken", "fix the", "fix this",
		"debug this", "troubleshoot", "failing test", "test fail", "test fails", "tests fail",
		"compile error", "type error", "runtime error", "crash", "exception", "returns 500",
		"still broken", "still failing",
	}
	for _, p := range fixPhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
