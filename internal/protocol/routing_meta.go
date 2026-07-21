package protocol

// Routing metadata keys stamped on agent response messages for observability.
const (
	MetadataRoutingProviderID                = "routing_provider_id"
	MetadataRoutingModel                     = "routing_model"
	MetadataRoutingToolModel                 = "routing_tool_model"
	MetadataRoutingReason                    = "routing_reason"
	MetadataRoutingSource                    = "routing_source"
	MetadataRoutingDomain                    = "routing_domain"
	MetadataRoutingCostTier                  = "routing_cost_tier"
	MetadataRoutingKnowledgeRoute            = "routing_knowledge_route"
	MetadataRoutingKnowledgeReason           = "routing_knowledge_reason"
	MetadataRoutingKnowledgeTargets          = "routing_knowledge_targets"
	MetadataRoutingKnowledgeExecuted         = "routing_knowledge_executed"
	MetadataRoutingComposerMode              = "routing_composer_mode"
	MetadataRoutingContextScope              = "routing_context_scope"
	MetadataRoutingImplSession               = "routing_impl_session"
	MetadataRoutingClassifierIntent          = "routing_classifier_intent"
	MetadataRoutingClassifierToolNeed        = "routing_classifier_tool_need"
	MetadataRoutingClassifierConfidence      = "routing_classifier_confidence"
	MetadataRoutingClassifierLoRATag         = "routing_classifier_lora_tag"
	MetadataRoutingConversationTier          = "routing_conversation_tier"
	MetadataRoutingConversationReasons       = "routing_conversation_reasons"
	MetadataRoutingConversationEscalatedFrom = "routing_conversation_escalated_from"
	MetadataTraceID                          = "trace_id"
	MetadataTraceSpans                       = "trace_spans"
)

// RoutingMeta captures per-turn model routing decisions for UI display.
type RoutingMeta struct {
	ProviderID                string   `json:"provider_id,omitempty"`
	Model                     string   `json:"model,omitempty"`
	ToolModel                 string   `json:"tool_model,omitempty"`
	Reason                    string   `json:"reason,omitempty"`
	Source                    string   `json:"source,omitempty"`
	Domain                    string   `json:"domain,omitempty"`
	CostTier                  string   `json:"cost_tier,omitempty"`
	KnowledgeRoute            string   `json:"knowledge_route,omitempty"`
	KnowledgeReason           string   `json:"knowledge_reason,omitempty"`
	KnowledgeTargets          []string `json:"knowledge_targets,omitempty"`
	KnowledgeExecuted         []string `json:"knowledge_executed,omitempty"`
	ComposerMode              string   `json:"composer_mode,omitempty"`
	ContextScope              string   `json:"context_scope,omitempty"`
	ImplSession               bool     `json:"impl_session,omitempty"`
	ClassifierIntent          string   `json:"classifier_intent,omitempty"`
	ClassifierToolNeed        bool     `json:"classifier_tool_need,omitempty"`
	ClassifierConfidence      float64  `json:"classifier_confidence,omitempty"`
	ClassifierLoRATag         string   `json:"classifier_lora_tag,omitempty"`
	ConversationTier          string   `json:"conversation_tier,omitempty"`
	ConversationReasons       []string `json:"conversation_reasons,omitempty"`
	ConversationEscalatedFrom string   `json:"conversation_escalated_from,omitempty"`
}

// ApplyRoutingMeta writes routing fields onto message metadata.
func ApplyRoutingMeta(msg *Message, meta RoutingMeta) {
	if msg == nil {
		return
	}
	if meta.ProviderID == "" && meta.Model == "" && meta.Reason == "" && meta.KnowledgeRoute == "" &&
		len(meta.KnowledgeTargets) == 0 && meta.ConversationTier == "" {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	if meta.ProviderID != "" {
		msg.Metadata[MetadataRoutingProviderID] = meta.ProviderID
	}
	if meta.Model != "" {
		msg.Metadata[MetadataRoutingModel] = meta.Model
	}
	if meta.ToolModel != "" {
		msg.Metadata[MetadataRoutingToolModel] = meta.ToolModel
	}
	if meta.Reason != "" {
		msg.Metadata[MetadataRoutingReason] = meta.Reason
	}
	if meta.Source != "" {
		msg.Metadata[MetadataRoutingSource] = meta.Source
	}
	if meta.Domain != "" {
		msg.Metadata[MetadataRoutingDomain] = meta.Domain
	}
	if meta.CostTier != "" {
		msg.Metadata[MetadataRoutingCostTier] = meta.CostTier
	}
	if meta.KnowledgeRoute != "" {
		msg.Metadata[MetadataRoutingKnowledgeRoute] = meta.KnowledgeRoute
	}
	if meta.KnowledgeReason != "" {
		msg.Metadata[MetadataRoutingKnowledgeReason] = meta.KnowledgeReason
	}
	if len(meta.KnowledgeTargets) > 0 {
		msg.Metadata[MetadataRoutingKnowledgeTargets] = meta.KnowledgeTargets
	}
	if len(meta.KnowledgeExecuted) > 0 {
		msg.Metadata[MetadataRoutingKnowledgeExecuted] = meta.KnowledgeExecuted
	}
	if meta.ComposerMode != "" {
		msg.Metadata[MetadataRoutingComposerMode] = meta.ComposerMode
	}
	if meta.ContextScope != "" {
		msg.Metadata[MetadataRoutingContextScope] = meta.ContextScope
	}
	if meta.ImplSession {
		msg.Metadata[MetadataRoutingImplSession] = true
	}
	if meta.ClassifierIntent != "" {
		msg.Metadata[MetadataRoutingClassifierIntent] = meta.ClassifierIntent
	}
	if meta.ClassifierToolNeed {
		msg.Metadata[MetadataRoutingClassifierToolNeed] = true
	}
	if meta.ClassifierConfidence > 0 {
		msg.Metadata[MetadataRoutingClassifierConfidence] = meta.ClassifierConfidence
	}
	if meta.ClassifierLoRATag != "" {
		msg.Metadata[MetadataRoutingClassifierLoRATag] = meta.ClassifierLoRATag
	}
	if meta.ConversationTier != "" {
		msg.Metadata[MetadataRoutingConversationTier] = meta.ConversationTier
	}
	if len(meta.ConversationReasons) > 0 {
		msg.Metadata[MetadataRoutingConversationReasons] = meta.ConversationReasons
	}
	if meta.ConversationEscalatedFrom != "" {
		msg.Metadata[MetadataRoutingConversationEscalatedFrom] = meta.ConversationEscalatedFrom
	}
}

// ExtractRoutingMeta reads routing metadata from a message.
func ExtractRoutingMeta(msg *Message) RoutingMeta {
	if msg == nil || msg.Metadata == nil {
		return RoutingMeta{}
	}
	out := RoutingMeta{}
	if v, ok := msg.Metadata[MetadataRoutingProviderID].(string); ok {
		out.ProviderID = v
	}
	if v, ok := msg.Metadata[MetadataRoutingModel].(string); ok {
		out.Model = v
	}
	if v, ok := msg.Metadata[MetadataRoutingToolModel].(string); ok {
		out.ToolModel = v
	}
	if v, ok := msg.Metadata[MetadataRoutingReason].(string); ok {
		out.Reason = v
	}
	if v, ok := msg.Metadata[MetadataRoutingSource].(string); ok {
		out.Source = v
	}
	if v, ok := msg.Metadata[MetadataRoutingDomain].(string); ok {
		out.Domain = v
	}
	if v, ok := msg.Metadata[MetadataRoutingCostTier].(string); ok {
		out.CostTier = v
	}
	if v, ok := msg.Metadata[MetadataRoutingKnowledgeRoute].(string); ok {
		out.KnowledgeRoute = v
	}
	if v, ok := msg.Metadata[MetadataRoutingKnowledgeReason].(string); ok {
		out.KnowledgeReason = v
	}
	out.KnowledgeTargets = stringSliceFromMeta(msg.Metadata[MetadataRoutingKnowledgeTargets])
	out.KnowledgeExecuted = stringSliceFromMeta(msg.Metadata[MetadataRoutingKnowledgeExecuted])
	if v, ok := msg.Metadata[MetadataRoutingComposerMode].(string); ok {
		out.ComposerMode = v
	}
	if v, ok := msg.Metadata[MetadataRoutingContextScope].(string); ok {
		out.ContextScope = v
	}
	if v, ok := msg.Metadata[MetadataRoutingImplSession].(bool); ok {
		out.ImplSession = v
	}
	if v, ok := msg.Metadata[MetadataRoutingClassifierIntent].(string); ok {
		out.ClassifierIntent = v
	}
	if v, ok := msg.Metadata[MetadataRoutingClassifierToolNeed].(bool); ok {
		out.ClassifierToolNeed = v
	}
	if v, ok := msg.Metadata[MetadataRoutingClassifierConfidence].(float64); ok {
		out.ClassifierConfidence = v
	} else if v, ok := msg.Metadata[MetadataRoutingClassifierConfidence].(int); ok {
		out.ClassifierConfidence = float64(v)
	}
	if v, ok := msg.Metadata[MetadataRoutingClassifierLoRATag].(string); ok {
		out.ClassifierLoRATag = v
	}
	if v, ok := msg.Metadata[MetadataRoutingConversationTier].(string); ok {
		out.ConversationTier = v
	}
	out.ConversationReasons = stringSliceFromMeta(msg.Metadata[MetadataRoutingConversationReasons])
	if v, ok := msg.Metadata[MetadataRoutingConversationEscalatedFrom].(string); ok {
		out.ConversationEscalatedFrom = v
	}
	return out
}

func stringSliceFromMeta(raw interface{}) []string {
	switch v := raw.(type) {
	case []string:
		if len(v) == 0 {
			return nil
		}
		return append([]string(nil), v...)
	case []interface{}:
		if len(v) == 0 {
			return nil
		}
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}
