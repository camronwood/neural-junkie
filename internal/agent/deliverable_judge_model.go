package agent

import (
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const deliverableJudgeMetadataKey = "deliverable_judge"
const judgeGeminiModelEnv = "NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"

func isDeliverableJudgeMessage(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	v, ok := msg.Metadata[deliverableJudgeMetadataKey]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(t)
		return strings.EqualFold(s, "true") || s == "1"
	default:
		return false
	}
}

func resolveDeliverableJudgeGeminiModel() string {
	return strings.TrimSpace(os.Getenv(judgeGeminiModelEnv))
}
