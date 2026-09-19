package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/fileedit"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEnrichNotFoundError_structuredPayload(t *testing.T) {
	t.Parallel()
	err := enrichNotFoundError(
		&fileedit.PatchError{Code: fileedit.ErrNotFound, Message: "old_string not found in file"},
		"line one\nhello world\nline three\n",
		"hello wrld",
	)
	pe, ok := err.(*fileedit.PatchError)
	if !ok {
		t.Fatalf("got %T", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(pe.JSONString()), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["code"] != fileedit.ErrNotFound || payload["fingerprint"] == "" {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestFileEditProposeOutcomeFromMessage_holdAndAuto(t *testing.T) {
	t.Parallel()
	held := protocol.NewMessage(protocol.MessageTypeFileChange, "c", protocol.AgentInfo{}, "x")
	held.Metadata[protocol.MetaFileChangeHeldForApproval] = true
	held.Metadata["file_change_hold_reason"] = "Held for approval: path policy blocked auto-apply for package.json"
	out := fileEditProposeOutcomeFromMessage(held, "package.json", nil)
	if out.Status != "pending_approval" {
		t.Fatalf("status=%q", out.Status)
	}
	if !strings.Contains(out.Reason, "path policy") {
		t.Fatalf("reason=%q", out.Reason)
	}
	payload := formatFileEditToolResult(out, "package.json", nil)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["status"] != "pending_approval" {
		t.Fatalf("tool payload=%s", payload)
	}

	auto := protocol.NewMessage(protocol.MessageTypeFileChange, "c", protocol.AgentInfo{}, "x")
	auto.Metadata[protocol.MetaFileChangeAutoApproved] = true
	out2 := fileEditProposeOutcomeFromMessage(auto, "src/a.go", nil)
	if out2.Status != "auto_approved" {
		t.Fatalf("status=%q", out2.Status)
	}
}

func TestRecordEditOutcome_snapshot(t *testing.T) {
	ResetEditOutcomeCounters()
	RecordEditOutcome("search_replace", "not_found", "", "")
	RecordEditOutcome("search_replace", "ok", "exact", "proposed")
	snap := GetEditOutcomeSnapshot()
	if snap.Counts["search_replace|not_found|-|-"] < 1 {
		t.Fatalf("counts=%v", snap.Counts)
	}
	if snap.Counts["search_replace|ok|exact|proposed"] < 1 {
		t.Fatalf("counts=%v", snap.Counts)
	}
}
