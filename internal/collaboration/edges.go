package collaboration

import (
	"fmt"
	"regexp"
	"strings"
)

// EvaluateEdgeCondition reports whether an incoming edge from upstream is satisfied.
func EvaluateEdgeCondition(cond *EdgeCondition, upstream CollaborationTask) (bool, error) {
	if cond == nil || cond.Mode == "" || cond.Mode == "always" {
		return upstream.Status == TaskCompleted, nil
	}
	switch cond.Mode {
	case "on_status":
		want := strings.TrimSpace(cond.Status)
		if want == "" {
			want = string(TaskCompleted)
		}
		return string(upstream.Status) == want, nil
	case "on_output":
		text := strings.TrimSpace(upstream.Output)
		if cond.Contains != "" {
			return strings.Contains(text, cond.Contains), nil
		}
		if cond.Regex != "" {
			re, err := regexp.Compile(cond.Regex)
			if err != nil {
				return false, fmt.Errorf("invalid edge regex: %w", err)
			}
			return re.MatchString(text), nil
		}
		return upstream.Status == TaskCompleted, nil
	default:
		return false, fmt.Errorf("unknown edge condition mode %q", cond.Mode)
	}
}

func isUpstreamSatisfiedForPolicy(dep CollaborationTask, policy BlockedUpstreamPolicy) bool {
	switch dep.Status {
	case TaskCompleted:
		return true
	case TaskBlocked:
		return policy == BlockedPolicySkipBranch
	default:
		return false
	}
}

// IsDependencySatisfied checks a single upstream task ID for readiness (unconditional dep).
func IsDependencySatisfied(depID string, tasks []CollaborationTask, policy BlockedUpstreamPolicy) bool {
	dep := TaskByID(tasks, depID)
	if dep == nil {
		return false
	}
	return isUpstreamSatisfiedForPolicy(*dep, policy)
}

// IsTaskReadyForCollab reports whether task can be dispatched given collaboration policy and edges.
func IsTaskReadyForCollab(task CollaborationTask, c *Collaboration) bool {
	if c == nil {
		return IsTaskReady(task, nil)
	}
	policy := c.EffectiveExecutionPolicy().BlockedUpstreamPolicy
	tasks := c.Tasks
	if len(task.Dependencies) == 0 && len(task.DependencyEdges) == 0 && len(task.DependencyGroups) == 0 {
		return true
	}
	edgeByFrom := make(map[string]*EdgeCondition, len(task.DependencyEdges))
	for i := range task.DependencyEdges {
		e := task.DependencyEdges[i]
		edgeByFrom[e.FromTaskID] = e.Condition
	}
	for _, depID := range task.Dependencies {
		dep := TaskByID(tasks, depID)
		if dep == nil {
			return false
		}
		if cond, ok := edgeByFrom[depID]; ok {
			okEdge, err := EvaluateEdgeCondition(cond, *dep)
			if err != nil || !okEdge {
				return false
			}
			continue
		}
		if !isUpstreamSatisfiedForPolicy(*dep, policy) {
			return false
		}
	}
	for _, e := range task.DependencyEdges {
		if containsString(task.Dependencies, e.FromTaskID) {
			continue
		}
		dep := TaskByID(tasks, e.FromTaskID)
		if dep == nil {
			return false
		}
		okEdge, err := EvaluateEdgeCondition(e.Condition, *dep)
		if err != nil || !okEdge {
			return false
		}
	}
	for _, g := range task.DependencyGroups {
		if len(g.TaskIDs) == 0 {
			continue
		}
		satisfied := 0
		for _, id := range g.TaskIDs {
			dep := TaskByID(tasks, id)
			if dep == nil {
				continue
			}
			if cond, ok := edgeByFrom[id]; ok {
				okEdge, err := EvaluateEdgeCondition(cond, *dep)
				if err == nil && okEdge {
					satisfied++
				}
				continue
			}
			if isUpstreamSatisfiedForPolicy(*dep, policy) {
				satisfied++
			}
		}
		mode := strings.ToLower(strings.TrimSpace(g.Mode))
		if mode == "any" {
			if satisfied == 0 {
				return false
			}
		} else if satisfied < len(g.TaskIDs) {
			return false
		}
	}
	return true
}

func containsString(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

// ReadyTasksForCollab returns pending, undispatched tasks ready under collaboration policy.
func ReadyTasksForCollab(c *Collaboration) []CollaborationTask {
	if c == nil {
		return nil
	}
	var out []CollaborationTask
	for _, t := range c.Tasks {
		if t.Status != TaskPending {
			continue
		}
		if t.PromptDispatched {
			continue
		}
		if t.Options != nil && t.Options.RequiresApproval && t.AwaitingApproval {
			continue
		}
		if !IsTaskReadyForCollab(t, c) {
			continue
		}
		out = append(out, t)
	}
	return out
}
