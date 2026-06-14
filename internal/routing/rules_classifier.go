package routing

// ClassifyRules returns a routing decision using keyword heuristics only.
func ClassifyRules(in Input) RoutingDecision {
	text := normText(in.Text)
	domain, domainReason := detectDomain(text, in.AgentType)
	toolNeed := toolNeedForAgentType(in.AgentType, text)
	costTier, costReason := detectCostTier(text, in.HasUserImages)

	reason := domainReason
	if in.HasUserImages {
		reason = costReason
	} else if toolNeed {
		reason = "tool_need_" + domainReason
	} else if costTier == CostCheap {
		reason = costReason
	} else if costTier == CostPremium && domain == DomainSecurity {
		reason = "security_premium"
	}

	tag, tagReason := selectLoRATag(in)
	if tag != "" {
		reason = tagReason
	} else if domain != DomainGeneral {
		tag, _ = selectLoRATag(Input{
			Text:          in.Text,
			AgentType:     domain,
			AgentModel:    in.AgentModel,
			InstalledTags: in.InstalledTags,
		})
		if tag == "" && domain != "" {
			reason = "domain_no_lora_installed"
		}
	}

	return RoutingDecision{
		Domain:     domain,
		ToolNeed:   toolNeed,
		CostTier:   costTier,
		Confidence: 1.0,
		Reason:     reason,
		Source:     SourceRules,
		LoRATag:    tag,
	}.Normalized()
}
