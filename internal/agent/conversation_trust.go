package agent

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/camronwood/neural-junkie/internal/ai"
	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ConversationTier controls model reliability for a conversation turn. It is
// deliberately separate from mutation/tool approval trust.
type ConversationTier string

const (
	ConversationTierStandard ConversationTier = "standard"
	ConversationTierElevated ConversationTier = "elevated"
	ConversationTierReliable ConversationTier = "reliable"
)

const (
	ConversationReasonExplicitToolAction = "explicit_tool_action"
	ConversationReasonLargeContext       = "large_context"
	ConversationReasonUserCorrection     = "user_correction"
	ConversationReasonRepeatedRequest    = "repeated_request"
	ConversationReasonQualityGateFailure = "quality_gate_failure"

	conversationQualityFailureKey   = "conversation_quality_gate_failure"
	conversationReliableReroutedKey = "conversation_reliable_rerouted"
	conversationLargeContextBytes   = 6 * 1024
	conversationContextScanNodes    = 256
)

// ConversationTrustDecision is the explainable model-routing trust decision.
type ConversationTrustDecision struct {
	Tier          ConversationTier
	Reasons       []string
	EscalatedFrom ConversationTier
}

var (
	explicitToolActionRE = regexp.MustCompile(`(?i)\b(?:run|execute|test|build|deploy|search|inspect|read|open|edit|write|create|delete|remove|rename|move|commit|push|apply|generate)\b`)
	userCorrectionRE     = regexp.MustCompile(`(?i)(?:\b(?:no|incorrect|actually|instead|you missed|you forgot|that is not|that's not|do not ask|correction)\b|\b(?:that is|that's|you are|you're)\s+wrong\b|\bwrong\s+(?:file|path|branch|model|provider|approach|answer)\b|\brename\s+(?:the\s+)?\w+\s+to\b|(?:^|\s)(?:again|as i (?:said|asked|requested))[,.:;!\s])`)
	repeatedRequestRE    = regexp.MustCompile(`(?i)\b(?:again|retry|try again|as i (?:said|asked|requested)|still need|already asked)\b`)
)

// ClassifyConversationTrust classifies a turn using conversation-only signals.
func (a *Agent) ClassifyConversationTrust(msg *protocol.Message) ConversationTrustDecision {
	decision := ConversationTrustDecision{Tier: ConversationTierStandard}
	if msg == nil {
		return decision
	}

	semanticDecision, hasSemanticDecision := protocol.ExtractTurnDecision(msg)
	if hasSemanticDecision {
		switch semanticDecision.Action {
		case semantic.ActionInspect, semantic.ActionDebug, semantic.ActionEdit, semantic.ActionRun, semantic.ActionContinue:
			decision.Reasons = append(decision.Reasons, ConversationReasonExplicitToolAction)
		}
		if semanticDecision.Interaction == semantic.InteractionCorrection {
			decision.Reasons = append(decision.Reasons, ConversationReasonUserCorrection)
		}
		if semanticDecision.Interaction == semantic.InteractionContinuation {
			decision.Reasons = appendUniqueString(decision.Reasons, ConversationReasonRepeatedRequest)
		}
	} else if explicitToolActionRE.MatchString(msg.Content) {
		decision.Reasons = append(decision.Reasons, ConversationReasonExplicitToolAction)
	}
	if len(msg.Content) >= conversationLargeContextBytes || metadataHasLargeContext(msg.Metadata) {
		decision.Reasons = append(decision.Reasons, ConversationReasonLargeContext)
	}
	if !hasSemanticDecision && userCorrectionRE.MatchString(msg.Content) {
		decision.Reasons = append(decision.Reasons, ConversationReasonUserCorrection)
	}
	if hasSemanticDecision {
		// Phrase-based "again/retry" matchers are rollback-only when no decision is stamped.
		if a.matchesPriorUserRequest(msg) {
			decision.Reasons = appendUniqueString(decision.Reasons, ConversationReasonRepeatedRequest)
		}
	} else if repeatedRequestRE.MatchString(msg.Content) || a.matchesPriorUserRequest(msg) {
		decision.Reasons = appendUniqueString(decision.Reasons, ConversationReasonRepeatedRequest)
	}
	if metadataBool(msg.Metadata, conversationQualityFailureKey) {
		decision.Reasons = append(decision.Reasons, ConversationReasonQualityGateFailure)
	}

	switch {
	case conversationContainsString(decision.Reasons, ConversationReasonQualityGateFailure),
		conversationContainsString(decision.Reasons, ConversationReasonUserCorrection),
		conversationContainsString(decision.Reasons, ConversationReasonRepeatedRequest):
		decision.Tier = ConversationTierReliable
	case len(decision.Reasons) > 0:
		decision.Tier = ConversationTierElevated
	}
	return decision
}

func (a *Agent) recordConversationTrust(provider ai.AIProvider, decision ConversationTrustDecision) {
	a.recordConversationTrustFor("", provider, decision)
}

func (a *Agent) recordConversationTrustFor(msgID string, provider ai.AIProvider, decision ConversationTrustDecision) {
	snap := RoutingSnapshot{
		ConversationTier:          string(decision.Tier),
		ConversationReasons:       append([]string(nil), decision.Reasons...),
		ConversationEscalatedFrom: string(decision.EscalatedFrom),
	}
	if provider != nil {
		snap.ProviderID = providerIDFromAI(provider)
		snap.ChatModel = strings.TrimSpace(provider.GetModel())
	}
	a.RecordRoutingSnapshotFor(msgID, snap)
}

func (a *Agent) matchesPriorUserRequest(msg *protocol.Message) bool {
	if a == nil || msg == nil {
		return false
	}
	current := normalizeConversationRequest(msg.Content)
	if len(current) < 12 {
		return false
	}
	history := a.channelHistory(msg.Channel)
	checked := 0
	for i := len(history) - 1; i >= 0 && checked < 8; i-- {
		prior := history[i]
		if prior == nil || prior.ID == msg.ID || !protocol.IsUserLikeSender(prior.From) {
			continue
		}
		checked++
		if normalizeConversationRequest(prior.Content) == current {
			return true
		}
	}
	return false
}

func normalizeConversationRequest(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

func metadataHasLargeContext(meta map[string]interface{}) bool {
	if meta == nil {
		return false
	}
	if raw, ok := meta["workspace_context"]; ok {
		if workspaceContextScope(meta, raw) == ContextScopeFull {
			return true
		}
		if boundedContextSize(raw, conversationLargeContextBytes, conversationContextScanNodes) >= conversationLargeContextBytes {
			return true
		}
	}
	for _, key := range []string{"prompt_attachments", "codebase_attachments", "context"} {
		switch v := meta[key].(type) {
		case string:
			if len(v) >= conversationLargeContextBytes {
				return true
			}
		case []byte:
			if len(v) >= conversationLargeContextBytes {
				return true
			}
		case []interface{}:
			if len(v) >= 4 {
				return true
			}
		}
	}
	return false
}

func workspaceContextScope(meta map[string]interface{}, raw interface{}) string {
	if scope, ok := meta[MetadataContextScope].(string); ok {
		return strings.ToLower(strings.TrimSpace(scope))
	}
	if ctxMap, ok := raw.(map[string]interface{}); ok {
		if scope, ok := ctxMap[MetadataContextScope].(string); ok {
			return strings.ToLower(strings.TrimSpace(scope))
		}
	}
	return ""
}

// boundedContextSize estimates serialized workspace payload size without
// marshaling or walking attacker-controlled metadata without a fixed bound.
func boundedContextSize(raw interface{}, byteLimit, nodeLimit int) int {
	if byteLimit <= 0 || nodeLimit <= 0 {
		return 0
	}
	size := 0
	nodes := 0
	var walk func(interface{})
	walk = func(value interface{}) {
		if size >= byteLimit || nodes >= nodeLimit {
			return
		}
		nodes++
		switch v := value.(type) {
		case string:
			size += min(len(v), byteLimit-size)
		case []byte:
			size += min(len(v), byteLimit-size)
		case map[string]interface{}:
			for key, item := range v {
				if size >= byteLimit || nodes >= nodeLimit {
					break
				}
				size += min(len(key), byteLimit-size)
				walk(item)
			}
		case []interface{}:
			for _, item := range v {
				if size >= byteLimit || nodes >= nodeLimit {
					break
				}
				walk(item)
			}
		case []map[string]interface{}:
			for _, item := range v {
				if size >= byteLimit || nodes >= nodeLimit {
					break
				}
				walk(item)
			}
		}
	}
	walk(raw)
	return size
}

func metadataBool(meta map[string]interface{}, key string) bool {
	v, _ := meta[key].(bool)
	return v
}

func appendUniqueString(values []string, value string) []string {
	if conversationContainsString(values, value) {
		return values
	}
	return append(values, value)
}

func conversationContainsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
