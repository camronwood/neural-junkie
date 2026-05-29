package agent

import (
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ApplyCollaborationTaskMetadataOnReply sets task_id and task_status on agent replies.
func ApplyCollaborationTaskMetadataOnReply(responseMsg *protocol.Message, source *protocol.Message, responseContent string) {
	if responseMsg == nil || source == nil {
		return
	}
	collabID := source.GetCollaborationID()
	if collabID == "" {
		return
	}
	responseMsg.SetCollaborationID(collabID)
	if phase := source.GetCollaborationPhase(); phase != "" {
		responseMsg.SetCollaborationPhase(phase)
	}
	taskID := source.GetTaskID()
	if taskID == "" {
		return
	}
	responseMsg.SetTaskID(taskID)

	if inferred := collaboration.InferTaskStatusFromAgentReply(responseContent, false); inferred != "" {
		responseMsg.SetTaskStatus(string(inferred))
		if output := source.GetTaskOutput(); output != "" {
			responseMsg.SetTaskOutput(output)
		}
		return
	}
	if taskStatus := source.GetTaskStatus(); taskStatus != "" && source.Type != protocol.MessageTypeCollabTask {
		responseMsg.SetTaskStatus(taskStatus)
	}
	if taskOutput := source.GetTaskOutput(); taskOutput != "" {
		responseMsg.SetTaskOutput(taskOutput)
	}
}

// CollaborationExecutionTaskStatusInstructions returns prompt text for execution-phase task reporting.
func CollaborationExecutionTaskStatusInstructions() string {
	return "Execution replies must produce work, not plan discussion.\n" +
		"When the task requires a document or code change, emit [FILE_CHANGE] with paths relative to the project root (under collabs/<collab-id>/ when a deliverables folder is set).\n" +
		"Shell commands belong in ```bash blocks only when they are real commands (not bare filenames like findings.md).\n" +
		"When your assigned task is finished, end with: TASK_STATUS: completed — plus a one-line summary of what you shipped (files, paths, conclusions).\n" +
		"If blocked, use TASK_STATUS: blocked and explain why.\n" +
		"Do not leave work marked pending if it is done; chat-only summaries do not complete the task.\n" +
		"Do not ask the user to run /approve-plan or re-open planning — execution has already started.\n"
}
