package collaboration

import "testing"

func TestValidateTaskIRs(t *testing.T) {
	errs := ValidateTaskIRs([]CollaborationTaskIR{{Title: "ab"}})
	if len(errs) == 0 {
		t.Fatal("expected title too short error")
	}
	errs = ValidateTaskIRs([]CollaborationTaskIR{{ID: "t1", Title: "Write API docs"}})
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
}
