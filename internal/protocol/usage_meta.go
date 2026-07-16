package protocol

// Inference usage metadata keys stamped on agent response messages.
const (
	MetadataInferenceUsage     = "inference_usage"
	MetadataPromptTokens       = "prompt_tokens"
	MetadataCompletionTokens   = "completion_tokens"
	MetadataEstimatedCostUSD   = "estimated_cost_usd"
)

// ApplyInferenceUsageMeta writes token/cost fields onto message metadata.
func ApplyInferenceUsageMeta(msg *Message, usage map[string]interface{}) {
	if msg == nil || len(usage) == 0 {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[MetadataInferenceUsage] = usage
	if v, ok := usage["prompt_tokens"]; ok {
		msg.Metadata[MetadataPromptTokens] = v
	}
	if v, ok := usage["completion_tokens"]; ok {
		msg.Metadata[MetadataCompletionTokens] = v
	}
	if v, ok := usage["estimated_cost_usd"]; ok {
		msg.Metadata[MetadataEstimatedCostUSD] = v
	}
}
