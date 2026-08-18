package intent

import "strings"

// DeriveContextPlan builds the authoritative context plan from the finalized
// decision action/retrieval/reason_codes and optional classifier proposals.
func DeriveContextPlan(features TurnFeatures, decision TurnDecision, semantic SemanticIntent) (ContextPlan, bool) {
	plan := ContextPlan{
		Tier:       ContextTier(strings.TrimSpace(string(semantic.ContextTier))),
		Subject:    ContextSubject(strings.TrimSpace(string(semantic.Subject))),
		ReviewMode: ReviewMode(strings.TrimSpace(string(semantic.ReviewMode))),
	}
	usedFallback := false
	if plan.Tier == "" || plan.Subject == "" || plan.ReviewMode == "" {
		usedFallback = true
		fallback := deriveContextPlanFallback(features, decision)
		if plan.Tier == "" {
			plan.Tier = fallback.Tier
		}
		if plan.Subject == "" {
			plan.Subject = fallback.Subject
		}
		if plan.ReviewMode == "" {
			plan.ReviewMode = fallback.ReviewMode
		}
		if len(plan.RequestedCategories) == 0 {
			plan.RequestedCategories = fallback.RequestedCategories
		}
		if len(plan.RequestedCapabilities) == 0 {
			plan.RequestedCapabilities = fallback.RequestedCapabilities
		}
		plan.ActiveContentSufficient = fallback.ActiveContentSufficient
	}
	plan = normalizeContextPlan(features, decision, plan)
	return plan, usedFallback
}

func deriveContextPlanFallback(features TurnFeatures, decision TurnDecision) ContextPlan {
	reasons := decision.ReasonCodes
	hasCodebase := containsRetrievalTarget(decision.Retrieval, RetrievalCodebase)
	plan := ContextPlan{
		Tier:       ContextTierNone,
		Subject:    ContextSubjectConversation,
		ReviewMode: ReviewModeNone,
	}
	if !features.HasWorkspace {
		return plan
	}

	switch {
	case hasReasonCode(reasons, "project_overview") ||
		hasReasonCode(reasons, "workspace_report") ||
		(decision.Action == ActionInspect && hasCodebase &&
			(strings.EqualFold(decision.Domain, "code_review") ||
				strings.EqualFold(decision.RecipientType, "code-review"))):
		plan.Tier = ContextTierOutline
		plan.Subject = ContextSubjectWorkspaceDocuments
		plan.ReviewMode = ReviewModeWorkspace
		plan.RequestedCategories = []string{"markdown", "docs", "readme"}
		plan.RequestedCapabilities = []string{"file_tree", "document_bodies"}
	case hasReasonCode(reasons, "repo_fact") || hasReasonCode(reasons, "git_inspect"):
		plan.Tier = ContextTierOutline
		plan.Subject = ContextSubjectCodebase
		plan.ReviewMode = ReviewModeNone
		plan.RequestedCapabilities = []string{"file_tree"}
		if hasReasonCode(reasons, "git_inspect") {
			plan.RequestedCapabilities = append(plan.RequestedCapabilities, "git_status")
		}
	case decision.Action == ActionInspect && hasCodebase:
		plan.Tier = ContextTierFocus
		plan.Subject = ContextSubjectActiveDocument
		plan.ReviewMode = ReviewModeDocument
		plan.ActiveContentSufficient = true
		plan.RequestedCapabilities = []string{"active_tab", "file_tree"}
	case decision.Action == ActionDebug || decision.Action == ActionEdit ||
		decision.Action == ActionRun || decision.Action == ActionContinue:
		plan.Tier = ContextTierFocus
		plan.Subject = ContextSubjectCodebase
		plan.ReviewMode = ReviewModeCode
		plan.RequestedCapabilities = []string{"file_tree", "active_tab", "diagnostics"}
	case decision.Action == ActionPlan && hasCodebase:
		plan.Tier = ContextTierOutline
		plan.Subject = ContextSubjectCodebase
		plan.ReviewMode = ReviewModeNone
		plan.RequestedCapabilities = []string{"file_tree"}
	case decision.Action == ActionArtifact && hasReasonCode(reasons, "workspace_report"):
		plan.Tier = ContextTierOutline
		plan.Subject = ContextSubjectWorkspaceDocuments
		plan.ReviewMode = ReviewModeWorkspace
		plan.RequestedCategories = []string{"markdown", "docs", "readme"}
		plan.RequestedCapabilities = []string{"file_tree", "document_bodies"}
	default:
		if features.HasWorkspace {
			plan.Tier = ContextTierHint
			plan.Subject = ContextSubjectConversation
			plan.ReviewMode = ReviewModeNone
		}
	}
	return plan
}

func normalizeContextPlan(features TurnFeatures, decision TurnDecision, plan ContextPlan) ContextPlan {
	if !features.HasWorkspace {
		plan.Tier = ContextTierNone
		if plan.Subject == "" {
			plan.Subject = ContextSubjectConversation
		}
		if plan.ReviewMode == "" {
			plan.ReviewMode = ReviewModeNone
		}
		plan.RequestedCategories = nil
		plan.RequestedCapabilities = nil
		plan.ActiveContentSufficient = false
		return plan
	}
	if plan.Tier == "" {
		plan.Tier = ContextTierHint
	}
	if plan.Subject == "" {
		plan.Subject = ContextSubjectConversation
	}
	if plan.ReviewMode == "" {
		plan.ReviewMode = ReviewModeNone
	}
	if plan.Subject == ContextSubjectWorkspaceDocuments && plan.ReviewMode == ReviewModeNone {
		plan.ReviewMode = ReviewModeWorkspace
	}
	if plan.Subject == ContextSubjectActiveDocument && plan.ReviewMode == ReviewModeNone {
		plan.ReviewMode = ReviewModeDocument
	}
	if decision.Action == ActionInspect &&
		(strings.EqualFold(decision.Domain, "code_review") ||
			strings.EqualFold(decision.RecipientType, "code-review")) {
		plan.Subject = ContextSubjectCodebase
		plan.ReviewMode = ReviewModeCode
		if plan.Tier == ContextTierHint || plan.Tier == ContextTierNone {
			plan.Tier = ContextTierOutline
		}
	}
	plan.RequestedCategories = normalizeStrings(plan.RequestedCategories)
	plan.RequestedCapabilities = normalizeStrings(plan.RequestedCapabilities)
	return plan
}

// ContextRequestFromPlan converts a stamp context plan into a client-facing
// prepare/fetch request (what content to upload next).
func ContextRequestFromPlan(plan ContextPlan, features TurnFeatures) ContextRequest {
	req := ContextRequest{
		Tier:                  plan.Tier,
		Subject:               plan.Subject,
		ReviewMode:            plan.ReviewMode,
		RequestedCategories:   append([]string(nil), plan.RequestedCategories...),
		RequestedCapabilities: append([]string(nil), plan.RequestedCapabilities...),
	}
	if !features.HasWorkspace || plan.Tier == ContextTierNone {
		return req
	}
	switch plan.Tier {
	case ContextTierHint:
		req.IncludeWorkspaceIdentity = true
	case ContextTierOutline:
		req.IncludeWorkspaceIdentity = true
		req.IncludeFileTree = true
		if plan.Subject == ContextSubjectWorkspaceDocuments {
			req.IncludeDocumentBodies = true
		}
	case ContextTierFocus:
		req.IncludeWorkspaceIdentity = true
		req.IncludeFileTree = true
		req.IncludeActiveTab = true
		req.IncludeSelection = true
		if plan.Subject == ContextSubjectWorkspaceDocuments {
			req.IncludeDocumentBodies = true
		}
	case ContextTierFull:
		req.IncludeWorkspaceIdentity = true
		req.IncludeFileTree = true
		req.IncludeActiveTab = true
		req.IncludeSelection = true
		req.IncludeDocumentBodies = true
		req.IncludeOpenFiles = true
	}
	for _, cap := range plan.RequestedCapabilities {
		switch strings.ToLower(strings.TrimSpace(cap)) {
		case "git_status":
			req.IncludeGitStatus = true
		case "diagnostics":
			req.IncludeDiagnostics = true
		case "document_bodies":
			req.IncludeDocumentBodies = true
		case "file_tree":
			req.IncludeFileTree = true
		case "active_tab":
			req.IncludeActiveTab = true
		}
	}
	return req
}

// ContextRequest is returned by /api/turn/prepare so the client uploads only
// the content the stamp needs.
type ContextRequest struct {
	Tier                     ContextTier   `json:"context_tier"`
	Subject                  ContextSubject `json:"subject"`
	ReviewMode               ReviewMode     `json:"review_mode"`
	RequestedCategories      []string       `json:"requested_categories,omitempty"`
	RequestedCapabilities    []string       `json:"requested_capabilities,omitempty"`
	IncludeWorkspaceIdentity bool           `json:"include_workspace_identity,omitempty"`
	IncludeFileTree          bool           `json:"include_file_tree,omitempty"`
	IncludeActiveTab         bool           `json:"include_active_tab,omitempty"`
	IncludeSelection         bool           `json:"include_selection,omitempty"`
	IncludeOpenFiles         bool           `json:"include_open_files,omitempty"`
	IncludeDocumentBodies    bool           `json:"include_document_bodies,omitempty"`
	IncludeGitStatus         bool           `json:"include_git_status,omitempty"`
	IncludeDiagnostics       bool           `json:"include_diagnostics,omitempty"`
}
