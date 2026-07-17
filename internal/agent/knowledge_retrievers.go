package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
	"github.com/camronwood/neural-junkie/internal/trace"
)

// KnowledgePhase is when a retriever runs in the turn pipeline.
type KnowledgePhase int

const (
	KnowledgePhaseEarly KnowledgePhase = iota
	KnowledgePhasePrompt
	KnowledgePhaseResponse
)

// KnowledgeResult describes a completed retrieval.
type KnowledgeResult struct {
	Executed bool
	Path     string
}

// KnowledgeRetriever is a pluggable knowledge source behind PlanKnowledgeRoute.
type KnowledgeRetriever interface {
	Target() routing.RouteTarget
	Phase() KnowledgePhase
	ShouldRun(plan routing.KnowledgePlan, msg *protocol.Message, intent TurnIntent) bool
	Execute(ctx context.Context, a *Agent, msg *protocol.Message, plan routing.KnowledgePlan, intent TurnIntent) (KnowledgeResult, error)
}

var knowledgeRetrievers = []KnowledgeRetriever{
	codebaseRetriever{},
	codeGraphRetriever{},
	learningsRetriever{},
	memoryRetriever{},
	priorReferenceRetriever{},
	repoConsultRetriever{},
}

// ExecuteKnowledgePlan runs registered retrievers for the given phase.
func (a *Agent) ExecuteKnowledgePlan(ctx context.Context, msg *protocol.Message, plan routing.KnowledgePlan, intent TurnIntent, phase KnowledgePhase) {
	if a == nil || msg == nil || skipKnowledgeRetrievalForMessage(msg) {
		return
	}
	for _, r := range knowledgeRetrievers {
		if r.Phase() != phase || !r.ShouldRun(plan, msg, intent) {
			continue
		}
		span := trace.StartSpan(ctx, "knowledge_execute."+string(r.Target()), map[string]any{
			"phase": phaseString(phase),
		})
		res, err := r.Execute(ctx, a, msg, plan, intent)
		if err != nil {
			span.EndError(err, nil)
			continue
		}
		span.End(map[string]any{"executed": res.Executed, "path": res.Path})
		if res.Executed && res.Path != "" {
			a.recordKnowledgeExecuted(res.Path)
		}
	}
}

func phaseString(p KnowledgePhase) string {
	switch p {
	case KnowledgePhaseEarly:
		return "early"
	case KnowledgePhasePrompt:
		return "prompt"
	case KnowledgePhaseResponse:
		return "response"
	default:
		return "unknown"
	}
}

type codebaseRetriever struct{}

func (codebaseRetriever) Target() routing.RouteTarget { return routing.RouteCodebase }
func (codebaseRetriever) Phase() KnowledgePhase       { return KnowledgePhaseEarly }
func (codebaseRetriever) ShouldRun(plan routing.KnowledgePlan, msg *protocol.Message, _ TurnIntent) bool {
	return ShouldRunCodebaseSearch(plan) || explicitCodebaseRequest(msg)
}
func (codebaseRetriever) Execute(_ context.Context, _ *Agent, msg *protocol.Message, plan routing.KnowledgePlan, _ TurnIntent) (KnowledgeResult, error) {
	if MergeCodebaseForRoute(msg, plan) {
		return KnowledgeResult{Executed: true, Path: "codebase"}, nil
	}
	if msg != nil && codebaseMentionRE.MatchString(msg.Content) {
		MergeCodebaseAttachments(msg)
		return KnowledgeResult{Executed: true, Path: "codebase"}, nil
	}
	return KnowledgeResult{}, nil
}

func explicitCodebaseRequest(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if msg.Metadata != nil {
		if v, ok := msg.Metadata["codebase_search"].(bool); ok && v {
			return true
		}
	}
	return false
}

type codeGraphRetriever struct{}

func (codeGraphRetriever) Target() routing.RouteTarget { return routing.RouteCodeGraph }
func (codeGraphRetriever) Phase() KnowledgePhase       { return KnowledgePhaseEarly }
func (codeGraphRetriever) ShouldRun(plan routing.KnowledgePlan, _ *protocol.Message, _ TurnIntent) bool {
	return plan.Has(routing.RouteCodeGraph)
}
func (codeGraphRetriever) Execute(_ context.Context, _ *Agent, msg *protocol.Message, plan routing.KnowledgePlan, _ TurnIntent) (KnowledgeResult, error) {
	if MergeGraphNeighborhoodForRoute(msg, plan) {
		return KnowledgeResult{Executed: true, Path: "code_graph"}, nil
	}
	return KnowledgeResult{}, nil
}

type memoryRetriever struct{}

func (memoryRetriever) Target() routing.RouteTarget { return routing.RouteConversationMemory }
func (memoryRetriever) Phase() KnowledgePhase       { return KnowledgePhasePrompt }
func (memoryRetriever) ShouldRun(plan routing.KnowledgePlan, _ *protocol.Message, _ TurnIntent) bool {
	return ShouldInjectMemory(plan)
}
func (memoryRetriever) Execute(_ context.Context, a *Agent, msg *protocol.Message, plan routing.KnowledgePlan, _ TurnIntent) (KnowledgeResult, error) {
	// Memory is injected during prompt build; mark execution when prompt path runs.
	_ = a
	_ = msg
	if ShouldInjectMemory(plan) {
		return KnowledgeResult{Executed: true, Path: knowledgeExecutedPathForMemory(plan)}, nil
	}
	return KnowledgeResult{}, nil
}

type learningsRetriever struct{}

func (learningsRetriever) Target() routing.RouteTarget { return routing.RouteLearning }
func (learningsRetriever) Phase() KnowledgePhase       { return KnowledgePhasePrompt }
func (learningsRetriever) ShouldRun(plan routing.KnowledgePlan, msg *protocol.Message, intent TurnIntent) bool {
	if skipKnowledgeRetrievalForMessage(msg) {
		return false
	}
	return plan.Has(routing.RouteLearning) || shouldInjectLearnings(msg, intent)
}
func (learningsRetriever) Execute(_ context.Context, a *Agent, msg *protocol.Message, _ routing.KnowledgePlan, _ TurnIntent) (KnowledgeResult, error) {
	if msg == nil {
		return KnowledgeResult{}, nil
	}
	_ = a
	return KnowledgeResult{Executed: true, Path: "learnings"}, nil
}

func shouldInjectLearnings(msg *protocol.Message, intent TurnIntent) bool {
	if intent == IntentClosure || intent == IntentLowSignal {
		return false
	}
	return msg != nil
}

type priorReferenceRetriever struct{}

func (priorReferenceRetriever) Target() routing.RouteTarget { return routing.RoutePriorReference }
func (priorReferenceRetriever) Phase() KnowledgePhase       { return KnowledgePhaseResponse }
func (priorReferenceRetriever) ShouldRun(plan routing.KnowledgePlan, _ *protocol.Message, _ TurnIntent) bool {
	return ShouldRunPriorReference(plan)
}
func (priorReferenceRetriever) Execute(_ context.Context, a *Agent, msg *protocol.Message, _ routing.KnowledgePlan, _ TurnIntent) (KnowledgeResult, error) {
	if !ShouldRunPriorReference(a.effectiveKnowledgePlan(msg, IntentSubstantive)) {
		return KnowledgeResult{}, nil
	}
	return KnowledgeResult{Executed: true, Path: "prior_reference"}, nil
}

type repoConsultRetriever struct{}

func (repoConsultRetriever) Target() routing.RouteTarget { return routing.RouteCodebase }
func (repoConsultRetriever) Phase() KnowledgePhase       { return KnowledgePhaseResponse }
func (repoConsultRetriever) ShouldRun(plan routing.KnowledgePlan, msg *protocol.Message, intent TurnIntent) bool {
	return aShouldRunRepoConsult(plan, msg, intent)
}
func (repoConsultRetriever) Execute(_ context.Context, a *Agent, msg *protocol.Message, _ routing.KnowledgePlan, intent TurnIntent) (KnowledgeResult, error) {
	if !a.shouldRunRepoConsult(context.Background(), msg, intent) {
		return KnowledgeResult{}, nil
	}
	return KnowledgeResult{Executed: true, Path: "repo_consult"}, nil
}

func aShouldRunRepoConsult(plan routing.KnowledgePlan, msg *protocol.Message, intent TurnIntent) bool {
	return ShouldRunCodebaseSearch(plan) && msg != nil && intent != IntentClosure
}
