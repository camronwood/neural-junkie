package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
)

func TestImplementationSessionState_RepeatedFailureCount(t *testing.T) {
	st := &ImplementationSessionState{}
	st.RecordCommandRun("make start-all", 2, "exit_code=2\nNo rule to make target 'start-all'")
	st.RecordCommandRun("make start-all", 2, "exit_code=2\nNo rule to make target 'start-all'")
	if got := st.repeatedFailureCount("make start-all"); got != 2 {
		t.Fatalf("repeatedFailureCount = %d, want 2", got)
	}
	st.RecordReadPath("Makefile")
	st.RecordCommandRun("make start-all", 2, "exit_code=2\nfail")
	if got := st.repeatedFailureCount("make start-all"); got != 1 {
		t.Fatalf("after read, first failure count = %d, want 1", got)
	}
	st.RecordCommandRun("make start-all", 2, "exit_code=2\nfail")
	if got := st.repeatedFailureCount("make start-all"); got != 2 {
		t.Fatalf("after read, repeatedFailureCount = %d, want 2", got)
	}
}

func TestImplementationSessionState_ShouldBlockRunCommand(t *testing.T) {
	st := &ImplementationSessionState{BootFixIntent: true}
	if err := st.ShouldBlockRunCommand("make start-all"); err == nil {
		t.Fatal("expected boot-fix read gate before reads")
	}
	st.RecordReadPath("Makefile")
	if err := st.ShouldBlockRunCommand("make start-all"); err != nil {
		t.Fatalf("unexpected block after read: %v", err)
	}
	st.RecordCommandRun("make start-all", 2, "exit_code=2\nfail")
	st.RecordCommandRun("make start-all", 2, "exit_code=2\nfail")
	if err := st.ShouldBlockRunCommand("make start-all"); err == nil {
		t.Fatal("expected repeated failure block")
	}
	if !st.CircuitBreakerTriggered() {
		t.Fatal("expected circuit breaker flag")
	}
}

func TestCommandOutputMatchesPlaybook(t *testing.T) {
	if got := commandOutputMatchesPlaybook("make: *** No rule to make target 'start-all'.  Stop."); got != "missing_start_all_target" {
		t.Fatalf("got %q", got)
	}
}

func TestSynthesizeMakefileWithStartAll(t *testing.T) {
	body := synthesizeMakefileWithStartAll("help:\n\t@echo ok\n")
	if !containsAllParts(body, "start-all:", "scripts/start-all.sh", ".PHONY") {
		t.Fatalf("missing start-all target: %q", body)
	}
}

func containsAllParts(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsStr(s, p) {
			return false
		}
	}
	return true
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestBootFixReadGate(t *testing.T) {
	if !shared.BootFixBootCommand("make start-all") {
		t.Fatal("make start-all should be boot command")
	}
	if shared.BootFixBootCommand("npm test") {
		t.Fatal("npm test should not be boot gated")
	}
}
