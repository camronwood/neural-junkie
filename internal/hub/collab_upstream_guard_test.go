package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeUpdateTaskStatus_blocksCompletionWhenUpstreamOpen(t *testing.T) {
	h := newTestHub(t)
	chName := "upstream-guard"
	_ = h.CreateChannel(chName, "test", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "PlatformEngineer", Type: protocol.AgentTypeDevOps, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("synthesis guard", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _ = cm.EnsureExecutionTasks(collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "Identify files", AssignedTo: "a1", AssignedName: "PlatformEngineer", Status: collaboration.TaskInProgress},
		{ID: "t2", Title: "Compile findings from the above tasks", AssignedTo: "a2", AssignedName: "Assistant", Status: collaboration.TaskPending, Dependencies: []string{"t1"}},
	}
	_ = cm.SetTasks(collab.ID, tasks)

	reply := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a2, "Done.\nTASK_STATUS: completed\n")
	reply.SetCollaborationID(collab.ID)
	reply.SetTaskID("t2")

	h.maybeUpdateTaskStatus(reply, collab.ID)

	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Tasks[1].Status == collaboration.TaskCompleted {
		t.Fatalf("expected synthesis task to stay open while upstream in progress, got %s", after.Tasks[1].Status)
	}
}
