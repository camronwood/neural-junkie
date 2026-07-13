package collaboration

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/schema"
)

// CollaborationTaskIR is the validated intermediate representation for plan/runbook tasks.
type CollaborationTaskIR struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	AssignedName string   `json:"assigned_name,omitempty"`
	AssignedTo   string   `json:"assigned_to,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// ValidateCollaborationTaskIR returns schema errors for a parsed task row.
func ValidateCollaborationTaskIR(t CollaborationTaskIR) []error {
	var errs []error
	title := strings.TrimSpace(t.Title)
	if title == "" {
		errs = append(errs, &schema.ValidationError{Path: "title", Message: "task title is required"})
	}
	if len(title) > 0 && len(title) < 3 {
		errs = append(errs, &schema.ValidationError{Path: "title", Message: "task title too short"})
	}
	for i, dep := range t.Dependencies {
		if strings.TrimSpace(dep) == "" {
			errs = append(errs, &schema.ValidationError{Path: fmt.Sprintf("dependencies[%d]", i), Message: "empty dependency id"})
		}
	}
	return errs
}

// TasksToIR maps CollaborationTask rows to IR for validation.
func TasksToIR(tasks []CollaborationTask) []CollaborationTaskIR {
	out := make([]CollaborationTaskIR, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, CollaborationTaskIR{
			ID:           t.ID,
			Title:        t.Title,
			Description:  t.Description,
			AssignedName: t.AssignedName,
			AssignedTo:   t.AssignedTo,
			Dependencies: append([]string(nil), t.Dependencies...),
		})
	}
	return out
}

// ValidateTaskIRs validates all tasks and returns combined errors.
func ValidateTaskIRs(tasks []CollaborationTaskIR) []error {
	var errs []error
	seen := map[string]struct{}{}
	for i, t := range tasks {
		for _, e := range ValidateCollaborationTaskIR(t) {
			if ve, ok := e.(*schema.ValidationError); ok {
				ve.Path = fmt.Sprintf("tasks[%d].%s", i, ve.Path)
				errs = append(errs, ve)
			} else {
				errs = append(errs, e)
			}
		}
		if t.ID != "" {
			if _, dup := seen[t.ID]; dup {
				errs = append(errs, &schema.ValidationError{Path: fmt.Sprintf("tasks[%d].id", i), Message: "duplicate task id"})
			}
			seen[t.ID] = struct{}{}
		}
	}
	return errs
}
