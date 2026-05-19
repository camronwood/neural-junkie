package collaboration

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

type runbookMockHub struct {
	agents map[string]*protocol.AgentInfo
}

func newRunbookMockHub() *runbookMockHub {
	return &runbookMockHub{agents: make(map[string]*protocol.AgentInfo)}
}

func (h *runbookMockHub) SendMessage(_ *protocol.Message) error { return nil }
func (h *runbookMockHub) GetAgent(id string) (*protocol.AgentInfo, error) {
	if a, ok := h.agents[id]; ok {
		return a, nil
	}
	return nil, nil
}
func (h *runbookMockHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) {
	var out []protocol.AgentInfo
	for _, a := range h.agents {
		out = append(out, *a)
	}
	return out, nil
}
func (h *runbookMockHub) CreateChannelWithType(name, _, _ string, _ protocol.ChannelType, _ string) *protocol.Channel {
	return &protocol.Channel{ID: uuid.New().String(), Name: name}
}
func (h *runbookMockHub) FindLiveAgentByDisplayName(string, protocol.AgentType) *protocol.AgentInfo {
	return nil
}

func (h *runbookMockHub) addAgent(id, name string, at protocol.AgentType, exp []string) {
	h.agents[id] = &protocol.AgentInfo{ID: id, Name: name, Type: at, Expertise: exp, Status: "active"}
}

func TestCreateRunbookDraftPhase(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "RustExpert", protocol.AgentTypeRust, []string{"rust"})
	cm := NewCollaborationManager(h)

	c, err := cm.CreateRunbook("Ship feature", []string{"a1"}, "general", "user", DiscussionConfig{}, CreateOptions{})
	if err != nil {
		t.Fatalf("CreateRunbook: %v", err)
	}
	if c.Phase != PhaseDraft {
		t.Fatalf("phase = %s, want draft", c.Phase)
	}
	if c.Source != SourceRunbook {
		t.Fatalf("source = %q", c.Source)
	}
	if c.Discussion != nil {
		t.Fatal("runbook should not start a planning discussion")
	}
}

func TestCreateRunbookAllowsSingleAgent(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "GoExpert", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	if _, err := cm.CreateRunbook("solo", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{}); err != nil {
		t.Fatalf("single-agent runbook: %v", err)
	}
}

func TestUpdateRunbookValidatesDAG(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	h.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("x", []string{"a1", "a2"}, "general", "u", DiscussionConfig{}, CreateOptions{})

	now := time.Now()
	t1 := CollaborationTask{ID: "t1", Title: "One", AssignedTo: "a1", Status: TaskPending, CreatedAt: now, UpdatedAt: now}
	t2 := CollaborationTask{ID: "t2", Title: "Two", AssignedTo: "a2", Status: TaskPending, Dependencies: []string{"t1", "t2"}, CreatedAt: now, UpdatedAt: now}

	_, err := cm.UpdateRunbook(c.ID, RunbookUpdatePayload{Tasks: []CollaborationTask{t1, t2}})
	if err == nil || !strings.Contains(err.Error(), "invalid task graph") {
		t.Fatalf("expected invalid graph error, got %v", err)
	}
}

func TestSubmitRunbookRequiresTasks(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	h.addAgent("a2", "B", protocol.AgentTypeSecurity, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("empty", []string{"a1", "a2"}, "general", "u", DiscussionConfig{}, CreateOptions{})

	if _, err := cm.SubmitRunbook(c.ID); err == nil || !strings.Contains(err.Error(), "at least one task") {
		t.Fatalf("expected task requirement, got %v", err)
	}
}

func TestSubmitRunbookDraftToReviewing(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "RustExpert", protocol.AgentTypeRust, []string{"rust"})
	h.addAgent("a2", "SecurityExpert", protocol.AgentTypeSecurity, []string{"security"})
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("rb", []string{"a1", "a2"}, "general", "u", DiscussionConfig{}, CreateOptions{})

	now := time.Now()
	tasks := []CollaborationTask{
		{ID: "t1", Title: "Scaffold", AssignedTo: "a1", AssignedName: "RustExpert", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "Review", AssignedTo: "a2", AssignedName: "SecurityExpert", Status: TaskPending, Dependencies: []string{"t1"}, CreatedAt: now, UpdatedAt: now},
	}
	if err := cm.SetTasks(c.ID, tasks); err != nil {
		t.Fatalf("SetTasks: %v", err)
	}
	out, err := cm.SubmitRunbook(c.ID)
	if err != nil {
		t.Fatalf("SubmitRunbook: %v", err)
	}
	if out.Phase != PhaseReviewing {
		t.Fatalf("phase = %s", out.Phase)
	}
}

func TestParsePlanTasksFromMarkdown(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "r1", AgentName: "RustExpert", AgentType: protocol.AgentTypeRust},
		{AgentID: "s1", AgentName: "SecurityExpert", AgentType: protocol.AgentTypeSecurity},
	}
	md := `## Plan
- Task 1: @RustExpert - Build
- Task 2: @SecurityExpert - Review
  - depends: 1
`
	tasks, err := ParsePlanTasks(md, agents)
	if err != nil {
		t.Fatalf("ParsePlanTasks: %v", err)
	}
	if len(tasks) != 2 || len(tasks[1].Dependencies) != 1 {
		t.Fatalf("tasks: %#v", tasks)
	}
}

func TestSuggestAssigneeSecurityTask(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("r1", "RustExpert", protocol.AgentTypeRust, []string{"rust"})
	h.addAgent("s1", "SecurityExpert", protocol.AgentTypeSecurity, []string{"auth", "encryption"})
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("rb", []string{"r1", "s1"}, "general", "u", DiscussionConfig{}, CreateOptions{})

	s, err := cm.SuggestRunbookAssignee(c.ID, "Threat model", "Review JWT and OAuth vulnerabilities")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil || s.AgentID != "s1" {
		t.Fatalf("expected security assignee, got %#v", s)
	}
}

func TestRunbookExecutionQAParticipantsAreTaskAssigneesOnly(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("r1", "RustExpert", protocol.AgentTypeRust, nil)
	h.addAgent("s1", "SecurityExpert", protocol.AgentTypeSecurity, nil)
	h.addAgent("g1", "Gemini", protocol.AgentTypeGeneral, nil)
	cm := NewCollaborationManager(h)

	c, err := cm.CreateRunbook("Research", []string{"r1", "s1", "g1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := cm.SetTasks(c.ID, []CollaborationTask{
		{ID: "t1", Title: "Rust work", AssignedTo: "r1", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
	}); err != nil {
		t.Fatal(err)
	}
	_, _ = cm.SubmitRunbook(c.ID)
	_, _ = cm.ApprovePlan(c.ID)
	out, err := cm.TransitionToExecuting(c.ID)
	if err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}
	if out.Discussion == nil {
		t.Fatal("expected execution discussion")
	}
	if len(out.Discussion.Participants) != 1 || out.Discussion.Participants[0] != "r1" {
		t.Fatalf("participants = %#v, want [r1] only", out.Discussion.Participants)
	}
}

func TestUpdateRunbookRejectsExecutingPhase(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	h.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("rb", []string{"a1", "a2"}, "general", "u", DiscussionConfig{}, CreateOptions{})

	now := time.Now()
	_ = cm.SetTasks(c.ID, []CollaborationTask{
		{ID: "t1", Title: "T", AssignedTo: "a1", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
	})
	_, _ = cm.SubmitRunbook(c.ID)
	_, _ = cm.ApprovePlan(c.ID)
	_, _ = cm.TransitionToExecuting(c.ID)

	_, err := cm.UpdateRunbook(c.ID, RunbookUpdatePayload{Description: "nope"})
	if err == nil || !strings.Contains(err.Error(), "draft or reviewing") {
		t.Fatalf("expected phase guard, got %v", err)
	}
}
