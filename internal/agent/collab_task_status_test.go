package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestApplyCollaborationTaskMetadataOnReplyTASKSTATUSOverridesPendingPrompt(t *testing.T) {
	source := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		"collab-ch",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@RustExpert your task",
	)
	source.SetCollaborationID("collab-1111-2222")
	source.SetCollaborationPhase("executing")
	source.SetTaskID("task-1")
	source.SetTaskStatus("pending")

	response := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		"collab-ch",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"Done.\nTASK_STATUS: completed\n",
	)

	ApplyCollaborationTaskMetadataOnReply(response, source, response.Content)

	if got := response.GetTaskStatus(); got != "completed" {
		t.Fatalf("task_status = %q, want completed", got)
	}
	if got := response.GetTaskID(); got != "task-1" {
		t.Fatalf("task_id = %q", got)
	}
}

func TestApplyCollaborationTaskMetadataOnReplyDoesNotEchoPendingFromTaskPrompt(t *testing.T) {
	source := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		"collab-ch",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@RustExpert your task",
	)
	source.SetCollaborationID("collab-1111-2222")
	source.SetTaskID("task-1")
	source.SetTaskStatus("pending")

	response := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		"collab-ch",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"Here is a partial update with no status keywords.",
	)

	ApplyCollaborationTaskMetadataOnReply(response, source, response.Content)

	if got := response.GetTaskStatus(); got != "" {
		t.Fatalf("task_status = %q, want empty (do not echo pending from collaboration_task)", got)
	}
}

func TestApplyCollaborationTaskMetadataOnReplyInfersInProgressFromContent(t *testing.T) {
	source := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		"collab-ch",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@RustExpert your task",
	)
	source.SetCollaborationID("collab-1111-2222")
	source.SetTaskID("task-1")
	source.SetTaskStatus("pending")

	response := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		"collab-ch",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"Still working on the layout.",
	)

	ApplyCollaborationTaskMetadataOnReply(response, source, response.Content)

	if got := response.GetTaskStatus(); got != "in_progress" {
		t.Fatalf("task_status = %q, want in_progress from content inference", got)
	}
}

func TestApplyCollaborationTaskMetadataOnReplyNoTaskIDSkipsStatus(t *testing.T) {
	source := protocol.NewMessage(
		protocol.MessageTypeChat,
		"collab-ch",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"TASK_STATUS: completed",
	)
	source.SetCollaborationID("collab-1111-2222")

	response := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		"collab-ch",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"TASK_STATUS: completed",
	)

	ApplyCollaborationTaskMetadataOnReply(response, source, response.Content)

	if response.GetTaskStatus() != "" {
		t.Fatalf("expected no task_status without task_id on source, got %q", response.GetTaskStatus())
	}
}
