package hub

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub/gitchange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestPendingUserQuestionRehydratesAfterRestart(t *testing.T) {
	t.Setenv(orchestrationDBEnv, filepath.Join(t.TempDir(), "orchestration.db"))
	first := NewHub()
	now := time.Now()
	question := &UserQuestion{
		ID: "question", AgentID: "agent", AgentName: "Agent", Channel: "general",
		Question: "Proceed?", Status: UserQuestionPending, CreatedAt: now,
	}
	first.persistUserQuestion(question, now.Add(time.Hour))
	if err := first.CloseOrchestrationStore(); err != nil {
		t.Fatal(err)
	}

	second := NewHub()
	defer second.CloseOrchestrationStore()
	second.RestoreDurableOrchestrationInputs()
	pending := second.userQuestionManager.ListPending()
	if len(pending) != 1 || pending[0].ID != question.ID {
		t.Fatalf("pending=%#v", pending)
	}
	if err := second.userQuestionManager.Answer(question.ID, "yes"); err != nil {
		t.Fatal(err)
	}
	if got := second.userQuestionManager.ListPending(); len(got) != 0 {
		t.Fatalf("question did not resolve: %#v", got)
	}
}

func TestPendingToolApprovalRehydratesAfterRestart(t *testing.T) {
	t.Setenv(orchestrationDBEnv, filepath.Join(t.TempDir(), "orchestration.db"))
	first := NewHub()
	now := time.Now()
	approval := &ToolApproval{
		ID: "approval", AgentID: "agent", AgentName: "Agent", SessionID: "session",
		ToolName: "write_file", Channel: "general", Status: ToolApprovalPending, CreatedAt: now,
	}
	first.persistToolApproval(approval, now.Add(time.Hour))
	if err := first.CloseOrchestrationStore(); err != nil {
		t.Fatal(err)
	}

	second := NewHub()
	defer second.CloseOrchestrationStore()
	second.RestoreDurableOrchestrationInputs()
	pending := second.toolApprovalManager.ListPending()
	if len(pending) != 1 || pending[0].ID != approval.ID {
		t.Fatalf("pending=%#v", pending)
	}
	if err := second.toolApprovalManager.Approve(approval.ID); err != nil {
		t.Fatal(err)
	}
	if got := second.toolApprovalManager.ListPending(); len(got) != 0 {
		t.Fatalf("approval did not resolve: %#v", got)
	}
}

func TestPendingFileAndGitApprovalsRehydrateAfterRestart(t *testing.T) {
	t.Setenv(orchestrationDBEnv, filepath.Join(t.TempDir(), "orchestration.db"))
	first := NewHub()
	now := time.Now()
	agent := protocol.AgentInfo{ID: "agent", Name: "Agent"}
	file := &filechange.FileChange{
		ID: "file-approval", Operation: filechange.FileOperationEdit, FilePath: "README.md",
		Agent: agent, Channel: "general", Status: filechange.FileChangeStatusPending,
		RequestedAt: now, ExpiresAt: now.Add(time.Hour),
		Metadata: map[string]any{"workspace_root": t.TempDir()},
	}
	git := &gitchange.Proposal{
		ID: "git-approval", Operation: gitchange.OpCommit, Agent: agent, Channel: "general",
		Status: "pending", RequestedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	first.persistFileChange(file)
	first.persistGitChange(git)
	if err := first.CloseOrchestrationStore(); err != nil {
		t.Fatal(err)
	}

	second := NewHub()
	defer second.CloseOrchestrationStore()
	second.RestoreDurableOrchestrationInputs()
	if pending := second.fileChangeManager.ListPendingFileChanges(""); len(pending) != 1 ||
		pending[0].ID != file.ID {
		t.Fatalf("file approvals=%#v", pending)
	}
	if pending := second.gitChangeManager.ListPending(""); len(pending) != 1 ||
		pending[0].ID != git.ID {
		t.Fatalf("git approvals=%#v", pending)
	}
}
