package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func collabTaskRoutingText(task collaboration.CollaborationTask) string {
	var b strings.Builder
	if title := strings.TrimSpace(task.Title); title != "" {
		b.WriteString(title)
	}
	if desc := strings.TrimSpace(task.Description); desc != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(desc)
	}
	return b.String()
}

func collabTaskRoutingOverrides(task collaboration.CollaborationTask) agent.TaskRoutingOverrides {
	var out agent.TaskRoutingOverrides
	if task.Options == nil {
		return out
	}
	out.ProviderID = task.Options.ProviderID
	return out
}

func assigneeInfoForTask(snap *collaboration.Collaboration, task collaboration.CollaborationTask) protocol.AgentInfo {
	if snap == nil {
		return protocol.AgentInfo{}
	}
	for _, a := range snap.Agents {
		if a.AgentID == task.AssignedTo {
			return protocol.AgentInfo{
				ID:   a.AgentID,
				Name: a.AgentName,
				Type: a.AgentType,
			}
		}
	}
	if task.AssignedName != "" {
		return protocol.AgentInfo{Name: task.AssignedName}
	}
	return protocol.AgentInfo{}
}

func (h *Hub) annotateCollaborationTaskRouting(snap *collaboration.Collaboration) {
	if snap == nil {
		return
	}
	router := agent.GlobalCollabRouting()
	if router == nil {
		return
	}
	ctx := context.Background()
	for i := range snap.Tasks {
		task := &snap.Tasks[i]
		if task.EffectiveKind() != collaboration.TaskKindAgent {
			continue
		}
		assignee := assigneeInfoForTask(snap, *task)
		if strings.TrimSpace(assignee.Name) == "" && strings.TrimSpace(assignee.ID) == "" {
			continue
		}
		plan := router.PlanTask(ctx, assignee, collabTaskRoutingText(*task), collabTaskRoutingOverrides(*task))
		if task.Options == nil {
			task.Options = &collaboration.TaskExecutionOptions{}
		}
		task.Options.ExpectedProviderID = strings.TrimSpace(plan.ProviderID)
		task.Options.ExpectedModel = strings.TrimSpace(plan.Model)
		task.Options.RoutingReason = strings.TrimSpace(plan.Reason)
	}
}

func formatCollabTaskRoutingNote(task collaboration.CollaborationTask) string {
	if task.Options == nil {
		return ""
	}
	model := strings.TrimSpace(task.Options.ExpectedModel)
	provider := strings.TrimSpace(task.Options.ExpectedProviderID)
	reason := strings.TrimSpace(task.Options.RoutingReason)
	if model == "" && provider == "" {
		return ""
	}
	var parts []string
	if model != "" {
		parts = append(parts, fmt.Sprintf("model `%s`", model))
	}
	if provider != "" {
		parts = append(parts, fmt.Sprintf("provider `%s`", provider))
	}
	note := "**Routing:** " + strings.Join(parts, ", ")
	if reason != "" {
		note += fmt.Sprintf(" (%s)", reason)
	}
	return "\n\n" + note
}

func applyCollabTaskRoutingMetadata(task collaboration.CollaborationTask, taskMsg *protocol.Message) {
	if taskMsg == nil || task.Options == nil {
		return
	}
	if taskMsg.Metadata == nil {
		taskMsg.Metadata = map[string]interface{}{}
	}
	if model := strings.TrimSpace(task.Options.ExpectedModel); model != "" {
		taskMsg.Metadata["task_expected_model"] = model
	}
	if provider := strings.TrimSpace(task.Options.ExpectedProviderID); provider != "" {
		taskMsg.Metadata["task_expected_provider_id"] = provider
	}
	if reason := strings.TrimSpace(task.Options.RoutingReason); reason != "" {
		taskMsg.Metadata["task_routing_reason"] = reason
	}
}
