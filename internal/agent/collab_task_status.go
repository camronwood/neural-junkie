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

	if inferred := collaboration.InferTaskStatusFromAgentReply(responseContent); inferred != "" {
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
	return "When your assigned task is finished, include a line: TASK_STATUS: completed (and a one-line summary).\n" +
		"If blocked, include TASK_STATUS: blocked and explain why.\n" +
		"Do not leave work marked pending if it is done.\n"
}
