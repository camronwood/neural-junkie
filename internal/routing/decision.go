// Package routing classifies tasks for domain, tool need, and cost tier routing.
package routing

import (
	"strings"

	semantic "github.com/camronwood/neural-junkie/internal/intent"
)

// Domain labels for specialist routing.
const (
	DomainGeneral      = "general"
	DomainSecurity     = "security"
	DomainBiology      = "biology"
	DomainFrontend     = "frontend"
	DomainBackend      = "backend"
	DomainDevOps       = "devops"
	DomainArchitecture = "architecture"
	DomainCodeReview   = "code_review"
	DomainDatabase     = "database"
	DomainRust         = "rust"
	DomainCAD          = "cad"
)

// Cost tier labels.
const (
	CostCheap    = "cheap"
	CostStandard = "standard"
	CostPremium  = "premium"
)

// Classifier source labels.
const (
	SourceRules = "rules"
	SourceLLM   = "llm"
)

// RoutingDecision is the unified output of task routing classifiers.
type RoutingDecision struct {
	Intent     string  `json:"intent,omitempty"`
	Domain     string  `json:"domain"`
	ToolNeed   bool    `json:"tool_need"`
	CostTier   string  `json:"cost_tier"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	Source     string  `json:"source"`
	LoRATag    string  `json:"lora_tag,omitempty"`
}

// Input is the feature context for routing classification.
type Input struct {
	Text            string
	AgentType       string
	AgentModel      string
	HasUserImages   bool
	InstalledTags   map[string]struct{}
	ConsultTriggers []string
	RepoPath        string
}

// Normalized returns a decision with defaults filled in.
func (d RoutingDecision) Normalized() RoutingDecision {
	out := d
	if out.Domain == "" {
		out.Domain = DomainGeneral
	}
	if out.CostTier == "" {
		out.CostTier = CostStandard
	}
	if out.Source == "" {
		out.Source = SourceRules
	}
	if out.Confidence < 0 {
		out.Confidence = 0
	} else if out.Confidence > 1 {
		out.Confidence = 1
	}
	return out
}

// DecisionFromSemantic adapts the canonical turn decision for provider and
// capability routing without classifying the raw user text again.
func DecisionFromSemantic(decision semantic.TurnDecision, installedTags map[string]struct{}) RoutingDecision {
	cost := CostStandard
	switch decision.Complexity {
	case "cheap":
		cost = CostCheap
	case "heavy":
		cost = CostPremium
	}
	toolNeed := false
	switch decision.Action {
	case semantic.ActionInspect, semantic.ActionDebug, semantic.ActionEdit,
		semantic.ActionRun, semantic.ActionContinue, semantic.ActionArtifact, semantic.ActionImage:
		toolNeed = true
	}
	out := RoutingDecision{
		Intent:     string(decision.Interaction),
		Domain:     strings.TrimSpace(decision.Domain),
		ToolNeed:   toolNeed,
		CostTier:   cost,
		Confidence: decision.Confidence,
		Reason:     "semantic_turn_decision",
		Source:     string(decision.Source),
	}
	if out.Domain == "" {
		out.Domain = DomainGeneral
	}
	out.LoRATag, _ = SelectLoRATag(Input{
		AgentType:     out.Domain,
		InstalledTags: installedTags,
	})
	return out
}
