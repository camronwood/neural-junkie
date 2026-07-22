package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func TestAppendMemoryForMessage_metadata(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.Open(dir + "/memory.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	memory.SetStore(store)
	memory.SetEnabledChecker(func() bool { return true })

	_ = store.UpsertChunk(memory.Chunk{
		ID: "msg:old", SourceType: memory.SourceMessage, SourceID: "old", Channel: "ch1",
		Content: "JWT refresh rotation", ContentHash: "h",
	})

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch1", protocol.AgentInfo{ID: "u", Name: "U", Type: "human"}, "JWT refresh")
	var sb strings.Builder
	plan := routing.PlanKnowledgeRoute(msg.Content)
	pr := AppendMemoryForMessage(&sb, msg, nil, plan)
	if pr.Count < 1 {
		t.Fatalf("expected injection, count=%d body=%q", pr.Count, sb.String())
	}
	if msg.Metadata["injected_memory_count"] == nil {
		t.Fatal("expected metadata")
	}
}

func TestAppendMemoryForMessage_closureSkips(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch1", protocol.AgentInfo{ID: "u", Name: "U", Type: "human"}, "thanks!")
	var sb strings.Builder
	plan := routing.PlanKnowledgeRoute(msg.Content)
	pr := AppendMemoryForMessage(&sb, msg, nil, plan)
	if pr.Count != 0 || sb.Len() > 0 {
		t.Fatalf("closure should skip memory injection, count=%d body=%q", pr.Count, sb.String())
	}
}

func TestBuildMemoryPromptContext_recoversThreadGoalAndCorrections(t *testing.T) {
	user := protocol.AgentInfo{ID: "u", Name: "U", Type: "human"}
	prior := protocol.NewMessage(protocol.MessageTypeQuestion, "ch1", user, "Use the legacy provider")
	prior.ID = "prior"
	correction := protocol.NewMessage(protocol.MessageTypeQuestion, "ch1", user, "No, use the configured provider instead")
	correction.ID = "correction"
	correction.ThreadID = "thread-1"
	correction.Metadata["original_goal_id"] = "goal-1"
	correction.Metadata["supersedes_message_ids"] = []interface{}{"older-prior"}

	pctx := buildMemoryPromptContext(correction, []*protocol.Message{prior}, routing.PlanKnowledgeRoute(correction.Content))
	if pctx.ThreadID != "thread-1" || pctx.GoalID != "goal-1" || !pctx.IsCorrection {
		t.Fatalf("context did not recover turn scope: %+v", pctx)
	}
	got := strings.Join(pctx.SupersededMessageIDs, ",")
	if !strings.Contains(got, "prior") || !strings.Contains(got, "older-prior") {
		t.Fatalf("superseded ids=%v", pctx.SupersededMessageIDs)
	}
}
