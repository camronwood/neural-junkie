package protocol

// Routing metadata keys stamped on agent response messages for observability.
const (
	MetadataRoutingProviderID = "routing_provider_id"
	MetadataRoutingModel      = "routing_model"
	MetadataRoutingToolModel  = "routing_tool_model"
	MetadataRoutingReason     = "routing_reason"
	MetadataRoutingSource     = "routing_source"
	MetadataRoutingDomain     = "routing_domain"
	MetadataRoutingCostTier       = "routing_cost_tier"
	MetadataRoutingKnowledgeRoute  = "routing_knowledge_route"
	MetadataRoutingKnowledgeReason = "routing_knowledge_reason"
)

// RoutingMeta captures per-turn model routing decisions for UI display.
type RoutingMeta struct {
	ProviderID string `json:"provider_id,omitempty"`
	Model      string `json:"model,omitempty"`
	ToolModel  string `json:"tool_model,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Source     string `json:"source,omitempty"`
	Domain          string `json:"domain,omitempty"`
	CostTier        string `json:"cost_tier,omitempty"`
	KnowledgeRoute  string `json:"knowledge_route,omitempty"`
	KnowledgeReason string `json:"knowledge_reason,omitempty"`
}

// ApplyRoutingMeta writes routing fields onto message metadata.
func ApplyRoutingMeta(msg *Message, meta RoutingMeta) {
	if msg == nil {
		return
	}
	if meta.ProviderID == "" && meta.Model == "" && meta.Reason == "" && meta.KnowledgeRoute == "" {
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
	return out
}
