package agent

import (
	"strings"

	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// stampedAction returns the hub TurnDecision action when present.
func stampedAction(msg *protocol.Message) (semantic.Action, bool) {
	if msg == nil {
		return "", false
	}
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok {
		return "", false
	}
	return decision.Action, true
}

func stampedDecision(msg *protocol.Message) (semantic.TurnDecision, bool) {
	if msg == nil {
		return semantic.TurnDecision{}, false
	}
	return protocol.ExtractTurnDecision(msg)
}

func decisionHasReason(decision semantic.TurnDecision, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, r := range decision.ReasonCodes {
		if strings.TrimSpace(r) == code {
			return true
		}
	}
	return false
}

func messageStampedArtifact(msg *protocol.Message) bool {
	action, ok := stampedAction(msg)
	return ok && action == semantic.ActionArtifact
}

func messageStampedImage(msg *protocol.Message) bool {
	action, ok := stampedAction(msg)
	return ok && action == semantic.ActionImage
}

func messageStampedMusic(msg *protocol.Message) bool {
	action, ok := stampedAction(msg)
	return ok && action == semantic.ActionMusic
}

func messageStampedMapsRoute(msg *protocol.Message) bool {
	decision, ok := stampedDecision(msg)
	if !ok {
		return false
	}
	return decision.Action == semantic.ActionArtifact && decisionHasReason(decision, "maps_route")
}

// messageStampedBootFailure reports whether the classifier tagged this turn as a boot/startup
// runtime failure via reason codes, rather than natural-language "won't boot" phrase matching.
func messageStampedBootFailure(msg *protocol.Message) bool {
	decision, ok := stampedDecision(msg)
	if !ok {
		return false
	}
	return decisionHasReason(decision, "runtime_failure") ||
		decisionHasReason(decision, "startup_failure") ||
		decisionHasReason(decision, "boot_failure")
}

func messageStampedImplAction(msg *protocol.Message) bool {
	action, ok := stampedAction(msg)
	if !ok {
		return false
	}
	switch action {
	case semantic.ActionDebug, semantic.ActionEdit, semantic.ActionContinue, semantic.ActionRun:
		return true
	default:
		return false
	}
}
