package collaboration

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteReviewAssetsWritesMarkdownFiles(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Now()
	c := &Collaboration{
		ID:               "collab-review-test",
		Title:            "Review test",
		Description:      "Persist review artifacts",
		Phase:            PhaseCompleted,
		ExecutionMode:    ExecutionModeSandbox,
		WorkingDirectory: "/tmp/collab-review-test",
		Plan: &SharedArtifact{
			ID:        "plan-1",
			Title:     "Collaboration Plan",
			Content:   "## Plan\n\n- Task 1: @Gemini - Review schemas",
			Version:   2,
			Status:    ArtifactApproved,
			CreatedAt: now,
			UpdatedAt: now,
		},
		PlanningRecap: "## Planning summary\n\n- Agreed on schema review.",
		SessionRecap:  "## Session summary\n\n- Completed the review.",
		Tasks: []CollaborationTask{
			{
				ID:           "task-1",
				Title:        "Review schemas",
				Description:  "Review schemas",
				AssignedName: "Gemini",
				Status:       TaskCompleted,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
	}

	paths, err := WriteReviewAssets(baseDir, c)
	if err != nil {
		t.Fatalf("WriteReviewAssets: %v", err)
	}

	assertFileContains(t, paths.Plan, "## Plan")
	assertFileContains(t, paths.PlanningSummary, "## Planning summary")
	assertFileContains(t, paths.SessionSummary, "## Session summary")
	assertFileContains(t, paths.Index, "collab-review-test")
	assertFileContains(t, paths.Index, ReviewAssetsPlanFileName)
	assertFileContains(t, paths.Index, ReviewAssetsPlanningSummaryName)
	assertFileContains(t, paths.Index, ReviewAssetsSessionSummaryName)
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q:\n%s", path, want, string(data))
	}
}
