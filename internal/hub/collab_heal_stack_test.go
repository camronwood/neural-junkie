package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Heal-stack integration tests (no Ollama): planning handoff, gen_error redispatch,
// premature TASK_STATUS without file, idle redispatch caps.

func TestHealStack_PlanningGenErrorThenDispatchHandoff(t *testing.T) {
	h := newTestHub(t)
	chName := "heal-planning-generr"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{
		MaxRounds:        1,
		MaxTotalMessages: 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	genErr := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		chName,
		*a1,
		"generation failed",
	)
	genErr.SetCollaborationID(collab.ID)
	genErr.SetCollaborationPhase(string(collaboration.PhasePlanning))
	genErr.Metadata = map[string]interface{}{"generation_error": true}
	if err := cm.RecordMessage(collab.ID, genErr); err != nil {
		t.Fatal(err)
	}

	if !NewCollabScheduler(h).OnPlanningNeedHandoff(collab.ID, "a1") {
		t.Fatal("expected OnPlanningNeedHandoff to succeed for current turn after gen_error")
	}
	msgs, _ := h.GetMessages(chName, 50)
	found := false
	for _, m := range msgs {
		if m == nil || !m.IsFromSystem() {
			continue
		}
		if m.Metadata != nil && m.Metadata["collab_turn_handoff"] == true && m.IsMentioned("a1") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected planning handoff with collab_turn_handoff metadata")
	}
}

func TestHealStack_GenerationErrorClearsPromptDispatched(t *testing.T) {
	h := newTestHub(t)
	chName := "heal-exec-generr"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	now := time.Now().Add(-time.Minute)
	taskID := "t-heal-generr-clear-dispatched"
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:               taskID,
			Title:            "Work",
			Description:      "Do work",
			AssignedTo:       "a1",
			AssignedName:     "AgentA",
			Status:           collaboration.TaskInProgress,
			PromptDispatched: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	})

	if err := cm.ClearTaskPromptDispatched(collab.ID, taskID); err != nil {
		t.Fatal(err)
	}
	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	for _, task := range snap.Tasks {
		if task.ID != taskID {
			continue
		}
		if task.PromptDispatched {
			t.Fatal("ClearTaskPromptDispatched must clear PromptDispatched for heal/redispatch")
		}
		if task.Status != collaboration.TaskPending {
			t.Fatalf("expected pending after clear, got %s", task.Status)
		}
		return
	}
	t.Fatal("task missing")
}

func TestHealStack_PrematureTaskStatusWithoutFile(t *testing.T) {
	h := newTestHub(t)
	chName := "heal-premature"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("write doc", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{}, collaboration.CreateOptions{
		SourceRepoPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	now := time.Now()
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:           "t-premature",
			Title:        "Write findings",
			Description:  "Write collabs/test/findings.md summarizing results",
			AssignedTo:   "a1",
			AssignedName: "AgentA",
			Status:       collaboration.TaskInProgress,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	})

	reply := protocol.NewMessage(protocol.MessageTypeAnswer, chName, *a1, "Done.\nTASK_STATUS: completed\n")
	reply.SetCollaborationID(collab.ID)
	reply.SetTaskID("t-premature")
	h.maybeUpdateTaskStatus(reply, collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	for _, task := range snap.Tasks {
		if task.ID != "t-premature" {
			continue
		}
		if task.Status == collaboration.TaskCompleted {
			t.Fatal("premature TASK_STATUS without FILE_CHANGE must not complete")
		}
		return
	}
	t.Fatal("task missing")
}

func TestHealStack_IdleRedispatchThenThrottle(t *testing.T) {
	h := newTestHub(t)
	chName := "heal-idle"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeAssistant, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	_, _, _ = cm.AcknowledgeWorkspace(collab.ID)

	dispatchedAt := time.Now().Add(-2 * collabIdleRedispatchAfter)
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:               "t1-idle-heal-stack-watchdog",
			Title:            "Work",
			Description:      "Do work",
			AssignedTo:       "a1",
			AssignedName:     "AgentA",
			Status:           collaboration.TaskPending,
			PromptDispatched: true,
			CreatedAt:        dispatchedAt,
			UpdatedAt:        dispatchedAt,
		},
	})

	before, _ := h.GetMessages(chName, 80)
	beforeCount := countMessageType(before, protocol.MessageTypeCollabTask)
	h.TickCollaborationIdleWatchdog(time.Now())
	mid, _ := h.GetMessages(chName, 80)
	midCount := countMessageType(mid, protocol.MessageTypeCollabTask)
	if midCount <= beforeCount {
		t.Fatalf("expected idle redispatch; before=%d mid=%d", beforeCount, midCount)
	}

	// Immediate second tick should be throttled / not flood unbounded.
	h.TickCollaborationIdleWatchdog(time.Now())
	after, _ := h.GetMessages(chName, 80)
	afterCount := countMessageType(after, protocol.MessageTypeCollabTask)
	if afterCount-midCount > 2 {
		t.Fatalf("unexpected flood of redispatches: mid=%d after=%d", midCount, afterCount)
	}
	_ = strings.TrimSpace(chName)
}
