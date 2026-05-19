package test

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHubCreateRunbookSession(t *testing.T) {
	h := hub.NewHub()
	h.CreateChannel("general", "General", "")
	registerTwoCollabAgents(t, h)

	result, err := h.CreateRunbookSession(hub.RunbookCreateRequest{
		Description: "Auth refactor runbook",
		AgentIDs:    []string{"a1", "a2"},
		Channel:     "general",
		CreatedBy:   "tester",
	})
	if err != nil {
		t.Fatalf("CreateRunbookSession: %v", err)
	}
	if result.CollaborationID == "" || result.CollaborationChannel == "" {
		t.Fatalf("missing ids: %#v", result)
	}
	if !strings.HasPrefix(result.CollaborationChannel, "collab-") {
		t.Fatalf("channel = %q", result.CollaborationChannel)
	}

	snap, err := h.GetRunbookSnapshot(result.CollaborationID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Phase != collaboration.PhaseDraft || snap.Source != collaboration.SourceRunbook {
		t.Fatalf("snap phase=%s source=%s", snap.Phase, snap.Source)
	}
}

func TestRunbookDAGExecutionViaHub(t *testing.T) {
	h := hub.NewHub()
	chName := "runbook-dag-exec"
	h.CreateChannel(chName, "rb", "")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SecurityExpert", Type: protocol.AgentTypeSecurity, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "GoExpert", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "A", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "B", AssignedTo: "a2", AssignedName: "SecurityExpert", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t3", Title: "C", AssignedTo: "a3", AssignedName: "GoExpert", Status: collaboration.TaskPending, Dependencies: []string{"t1", "t2"}, CreatedAt: now, UpdatedAt: now},
	}

	res, err := h.CreateRunbookSession(hub.RunbookCreateRequest{
		Description: "DAG runbook",
		AgentIDs:    []string{"a1", "a2", "a3"},
		Channel:     chName,
		CreatedBy:   "tester",
		Tasks: tasks,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	collabID := res.CollaborationID

	cm := h.GetCollaborationManager()
	if _, err := cm.SubmitRunbook(collabID); err != nil {
		t.Fatalf("submit: %v", err)
	}
	started, err := h.StartRunbook(collabID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Phase != collaboration.PhaseExecuting {
		t.Fatalf("phase = %s", started.Phase)
	}

	ch := res.CollaborationChannel
	_ = h.AcknowledgeCollaborationWorkspace(collabID, "")

	msgs, _ := h.GetMessages(ch, 200)
	wave1 := countCollabTaskMessages(msgs)
	if wave1 != 2 {
		t.Fatalf("wave1 want 2 collab_task after workspace ack, got %d", wave1)
	}

	reply1 := protocol.NewMessage(protocol.MessageTypeAnswer, ch, *a1, "Done.\nTASK_STATUS: completed\n")
	reply1.SetCollaborationID(collabID)
	reply1.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	reply1.SetTaskID("t1")
	_ = h.SendMessage(reply1)

	reply2 := protocol.NewMessage(protocol.MessageTypeAnswer, ch, *a2, "Done.\nTASK_STATUS: completed\n")
	reply2.SetCollaborationID(collabID)
	reply2.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	reply2.SetTaskID("t2")
	_ = h.SendMessage(reply2)

	msgs, _ = h.GetMessages(ch, 200)
	if countCollabTaskMessages(msgs) != 3 {
		t.Fatalf("wave2 want 3 collab_task total, got %d", countCollabTaskMessages(msgs))
	}
}

func countCollabTaskMessages(msgs []*protocol.Message) int {
	n := 0
	for _, m := range msgs {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			n++
		}
	}
	return n
}
