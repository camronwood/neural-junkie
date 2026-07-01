package collaboration

import (
	"strings"
	"testing"
)

func TestFormatPlanMarkdownAddsStructure(t *testing.T) {
	c := &Collaboration{
		Title:       "UI refresh",
		Description: "Update the dashboard",
		Plan: &SharedArtifact{
			Content: "Task 1: @Frontend - Write collabs/id/ui-state-spec.md",
		},
		Tasks: []CollaborationTask{
			{Title: "UI spec", AssignedName: "Frontend", Status: TaskPending},
		},
	}
	out := formatPlanMarkdown(c)
	if !strings.Contains(out, "# Plan") {
		t.Fatalf("missing plan header: %s", out)
	}
	if !strings.Contains(out, "**Goal:** Update the dashboard") {
		t.Fatalf("missing goal: %s", out)
	}
	if !strings.Contains(out, "- Task 1:") {
		t.Fatalf("missing bullet task: %s", out)
	}
	if !strings.Contains(out, "## Task Summary") {
		t.Fatalf("missing task summary: %s", out)
	}
}

func TestFormatPlanningSummaryMarkdownWrapsRecap(t *testing.T) {
	c := &Collaboration{
		Title:         "UI refresh",
		Description:   "Update the dashboard",
		PlanningRecap: "We agreed on a minimal task list focused on deliverables.",
	}
	out := formatPlanningSummaryMarkdown(c)
	if !strings.HasPrefix(out, "# Planning Summary") {
		t.Fatalf("missing header: %s", out)
	}
	if !strings.Contains(out, "## Discussion Recap") {
		t.Fatalf("missing recap section: %s", out)
	}
	if !strings.Contains(out, "minimal task list") {
		t.Fatalf("missing recap body: %s", out)
	}
}
