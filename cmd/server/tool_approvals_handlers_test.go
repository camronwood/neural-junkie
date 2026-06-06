package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestHandleToolApprovals_autoEditReadOnlyShell(t *testing.T) {
	chatHub = hub.NewHub()

	body, err := json.Marshal(map[string]interface{}{
		"agent_id":   "cursor-1",
		"agent_name": "Cursor",
		"session_id": "sess-1",
		"tool_name":  "run_shell_command",
		"tool_input": map[string]interface{}{"command": "cat collabs/abc/findings.md"},
		"channel":    "general",
		"mode":       "auto_edit",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tool-approvals", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleToolApprovals(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["decision"] != "allow" || resp["status"] != "approved" {
		t.Fatalf("expected auto-approve, got %#v", resp)
	}
}

func TestHandleToolApprovals_autoEditUnsafeShellPrompts(t *testing.T) {
	chatHub = hub.NewHub()

	body, err := json.Marshal(map[string]interface{}{
		"agent_id":   "cursor-1",
		"agent_name": "Cursor",
		"session_id": "sess-2",
		"tool_name":  "run_shell_command",
		"tool_input": map[string]interface{}{"command": "npm install"},
		"channel":    "general",
		"mode":       "auto_edit",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tool-approvals", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handleToolApprovals(rec, req)
		close(done)
	}()

	var pending []*hub.ToolApproval
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pending = chatHub.GetToolApprovalManager().ListPending()
		if len(pending) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(pending))
	}
	tam := chatHub.GetToolApprovalManager()
	tam.Reject(pending[0].ID, "test reject")
	<-done

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["decision"] != "deny" {
		t.Fatalf("expected deny, got %#v", resp)
	}
}

func TestHandleToolApprovals_autoEditFileTools(t *testing.T) {
	chatHub = hub.NewHub()

	body, err := json.Marshal(map[string]interface{}{
		"agent_id":   "cursor-1",
		"agent_name": "Cursor",
		"session_id": "sess-3",
		"tool_name":  "edit_file",
		"tool_input": map[string]interface{}{"path": "src/App.tsx"},
		"channel":    "general",
		"mode":       "auto_edit",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tool-approvals", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleToolApprovals(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["decision"] != "allow" {
		t.Fatalf("expected allow, got %#v", resp)
	}
}
