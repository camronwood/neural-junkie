package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// collabLane describes what an agent owns vs defers during collaboration.
type collabLane struct {
	owns   string
	defers string
	avoid  string
}

func collabLaneFor(agentType protocol.AgentType, agentName string) collabLane {
	nameLower := strings.ToLower(strings.TrimSpace(agentName))
	switch agentType {
	case protocol.AgentTypeArchitecture:
		return collabLane{
			owns:   "system/API boundaries, schema shape, registration approach, doc structure, standards, tradeoff decisions; assigned markdown/schema files via [FILE_CHANGE]",
			defers: "application code to backend/CLI specialists when not assigned; CI/CD and runtime ops to @PlatformEngineer or DevOps specialists",
			avoid:  "inventing k8s manifests or owning deployment runbooks unless assigned; duplicating peers' assigned deliverables",
		}
	case protocol.AgentTypeDevOps:
		return collabLane{
			owns:   "CI/CD, packaging, deployment, observability hooks, environment config, release mechanics; assigned pipeline/config files via [FILE_CHANGE]",
			defers: "API/schema design and markdown spec content to @SoftwareArchitect or architect types when not assigned",
			avoid:  "kubectl/helm/json tool-call payloads during planning when the goal is documentation or API schema; re-defining the schema model",
		}
	case protocol.AgentTypeCLI:
		if strings.Contains(nameLower, "gemini") {
			return collabLane{
				owns:   "drafting markdown/code from an approved outline, file-level implementation, refactors scoped to assigned tasks",
				defers: "architecture and schema registration policy to @SoftwareArchitect when not assigned; pipelines and deploy to @PlatformEngineer",
				avoid:  "re-opening plan negotiation or assigning tasks to other agents during execution",
			}
		}
		return collabLane{
			owns:   "implementation, patches, and concrete file deliverables from an approved plan",
			defers: "architecture and standards to architect agents when not assigned; infra and release to DevOps/platform agents",
			avoid:  "rewriting the whole plan each turn; duplicating tasks already owned by peers",
		}
	case protocol.AgentTypeBackend:
		return collabLane{
			owns:   "service/API handler design, data access patterns, backend contracts; assigned code and API files via [FILE_CHANGE]",
			defers: "pure doc/schema narrative to architects when not assigned; frontend UX to frontend agents; deploy to DevOps",
			avoid:  "owning OpenAPI prose end-to-end when an architect is assigned to that deliverable",
		}
	case protocol.AgentTypeAssistant:
		return collabLane{
			owns:   "clarifying the goal, synthesizing discussion, structuring tasks, sequencing work; assigned summary/deliverable files via [FILE_CHANGE]",
			defers: "deep schema design to @SoftwareArchitect when not assigned; infra to @PlatformEngineer when not assigned",
			avoid:  "generic filler tasks that duplicate peers; ignoring an assigned file deliverable",
		}
	case protocol.AgentTypeModerator:
		return collabLane{
			owns:   "keeping discussion on track, summarizing agreements, surfacing blockers",
			defers: "technical design and file deliverables to specialist agents unless explicitly assigned",
			avoid:  "assigning yourself technical deliverables without a task line",
		}
	case protocol.AgentTypeCodeReview:
		return collabLane{
			owns:   "review criteria, risk callouts, test gaps, merge readiness; assigned review notes via [FILE_CHANGE] when tasked",
			defers: "initial architecture to architects; fixes to the agent assigned the fix task",
			avoid:  "rewriting the plan from scratch",
		}
	case protocol.AgentTypeSecurity:
		return collabLane{
			owns:   "threat model, authn/z for APIs, sensitive-field handling; assigned security docs via [FILE_CHANGE]",
			defers: "general doc outline to architects when not assigned",
			avoid:  "duplicating architect's entire schema narrative",
		}
	case protocol.AgentTypeRepo:
		return collabLane{
			owns:   "repo-grounded analysis and assigned file deliverables under the project root via [FILE_CHANGE]",
			defers: "cross-cutting architecture to architect agents when not assigned",
			avoid:  "inventing paths not present in the workspace",
		}
	default:
		return collabLane{
			owns:   "contributions aligned with your domain expertise; assigned file deliverables via [FILE_CHANGE]",
			defers: "work clearly owned by other participants listed below",
			avoid:  "repeating the same tasks under different wording for multiple agents",
		}
	}
}

func appendCollaborationLaneInstructions(b *strings.Builder, collabInfo CollaborationInfo, self protocol.AgentInfo) {
	if b == nil || collabInfo.ID == "" {
		return
	}
	lane := collabLaneFor(self.Type, self.Name)
	b.WriteString("\n=== YOUR LANE (minimize overlap) ===\n")
	b.WriteString("**You own:** ")
	b.WriteString(lane.owns)
	b.WriteString("\n**Defer to peers for:** ")
	b.WriteString(lane.defers)
	b.WriteString("\n**Do not:** ")
	b.WriteString(lane.avoid)
	b.WriteString("\n")

	b.WriteString("\n=== PEER LANES (do not duplicate their work) ===\n")
	for _, ag := range collabInfo.Agents {
		if strings.EqualFold(ag.Name, self.Name) {
			continue
		}
		peer := collabLaneFor(protocol.AgentType(ag.Type), ag.Name)
		b.WriteString(fmt.Sprintf("- @%s (%s): owns %s\n", ag.Name, ag.Role, peer.owns))
	}
	b.WriteString("\nWhen assigning tasks in planning, give each task **one** primary assignee in that agent's lane. ")
	b.WriteString("Use @mentions for review or short input, not parallel duplicate deliverables.\n")
	if collabInfo.Phase == "executing" {
		b.WriteString("**File deliverables:** any assignee may author files with [FILE_CHANGE] in this IDE — you do not need a CLI agent to write assigned paths.\n")
	}
}
