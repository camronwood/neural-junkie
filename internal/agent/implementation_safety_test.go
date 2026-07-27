package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDestructiveRewriteGuard(t *testing.T) {
	old := strings.Repeat("const preserved = true;\n", 60)
	scaffold := "export default function App() {\n  return <main>Welcome</main>;\n}\n"
	risky, ratio := IsDestructiveFileRewrite(old, scaffold)
	if !risky || ratio <= destructiveRewriteRatio {
		t.Fatalf("expected destructive rewrite, risky=%v ratio=%.2f", risky, ratio)
	}

	lines := splitSafetyLines(old)
	lines[30] = "const preserved = false;"
	if risky, ratio := IsDestructiveFileRewrite(old, strings.Join(lines, "\n")+"\n"); risky {
		t.Fatalf("single-line edit classified destructive (ratio=%.2f)", ratio)
	}
}

func TestGitBaselineDetectsRewriteAfterWorkingTreeWasAlreadyTruncated(t *testing.T) {
	root := t.TempDir()
	path := "src/App.tsx"
	abs := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	baseline := strings.Repeat("const existingArchitecture = true;\n", 60)
	if err := os.WriteFile(abs, []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitSafetyTest(t, root, "init")
	runGitSafetyTest(t, root, "add", path)
	runGitSafetyTest(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "baseline")

	damaged := strings.Repeat("const placeholder = true;\n", 12)
	if err := os.WriteFile(abs, []byte(damaged), 0o644); err != nil {
		t.Fatal(err)
	}
	proposal := strings.Repeat("const anotherPlaceholder = true;\n", 11)
	if risky, _ := IsDestructiveFileRewrite(damaged, proposal); risky {
		t.Fatal("current-file guard should demonstrate the already-truncated baseline gap")
	}
	risky, ratio, baselineLines := gitBaselineRewriteRisk(context.Background(), root, path, proposal)
	if !risky || ratio <= destructiveRewriteRatio || baselineLines != 60 {
		t.Fatalf("Git baseline risk not detected: risky=%v ratio=%.2f lines=%d", risky, ratio, baselineLines)
	}
}

func runGitSafetyTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2000-01-01T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestDuplicateEditBreakerUsesResultingContentHash(t *testing.T) {
	state := &ImplementationSessionState{}
	if err := state.prepareEditSnapshot("", "src/App.tsx", "old", "new"); err != nil {
		t.Fatal(err)
	}
	state.recordEditResult("src/App.tsx", "new")
	err := state.prepareEditSnapshot("", "src/App.tsx", "newer old", "new")
	if err == nil || !strings.Contains(err.Error(), "exact resulting content") {
		t.Fatalf("expected duplicate result rejection, got %v", err)
	}
	if !state.CircuitBreakerFired {
		t.Fatal("duplicate edit must fire circuit breaker")
	}
}

func TestVerificationProgressGateStopsUnimprovedFailure(t *testing.T) {
	state := &ImplementationSessionState{}
	state.recordVerificationProgress("error: missing module\nerror: compile failed", true, false)
	if state.CircuitBreakerFired {
		t.Fatal("first verification establishes a baseline")
	}
	state.recordVerificationProgress("error: missing module\nerror: compile failed", true, false)
	if !state.CircuitBreakerFired || state.ConsecutiveNoVerifyProgress != 1 {
		t.Fatalf("expected no-progress breaker, state=%+v", state)
	}
}

func TestVerificationProgressGateAllowsLowerFailureCount(t *testing.T) {
	state := &ImplementationSessionState{}
	state.recordVerificationProgress("error: one\nerror: two", true, false)
	state.recordVerificationProgress("error: one", true, false)
	if state.CircuitBreakerFired || state.ConsecutiveNoVerifyProgress != 0 {
		t.Fatalf("improved verification should continue, state=%+v", state)
	}
}

func TestFailedAutoApplySessionRollsBackMatchingProposal(t *testing.T) {
	root := t.TempDir()
	path := "src/App.tsx"
	abs := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	original := strings.Repeat("const original = true;\n", 50)
	replacement := "export default () => <main>placeholder</main>;\n"
	if err := os.WriteFile(abs, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	if err := state.prepareEditSnapshot(root, path, original, replacement); err != nil {
		t.Fatal(err)
	}
	state.recordEditResult(path, replacement)
	if err := os.WriteFile(abs, []byte(replacement), 0o644); err != nil {
		t.Fatal(err)
	}
	state.VerifyFailed = true
	state.rollbackFailedAutoApplySession(root)
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original || len(state.RolledBackFiles) != 1 {
		t.Fatalf("rollback failed: files=%v content=%q", state.RolledBackFiles, got)
	}
}

func TestRollbackDoesNotClobberConcurrentUserEdit(t *testing.T) {
	root := t.TempDir()
	path := "App.tsx"
	abs := filepath.Join(root, path)
	if err := os.WriteFile(abs, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	if err := state.prepareEditSnapshot(root, path, "original", "agent edit"); err != nil {
		t.Fatal(err)
	}
	state.recordEditResult(path, "agent edit")
	if err := os.WriteFile(abs, []byte("user edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	state.VerifyFailed = true
	state.rollbackFailedAutoApplySession(root)
	got, _ := os.ReadFile(abs)
	if string(got) != "user edit" || len(state.RolledBackFiles) != 0 {
		t.Fatalf("concurrent edit was clobbered: files=%v content=%q", state.RolledBackFiles, got)
	}
}

func TestDestructiveRewriteTrustTiers(t *testing.T) {
	if CanAutoApproveDestructiveRewrite("ollama", nil) {
		t.Fatal("local model must not auto-approve a destructive rewrite")
	}
	if !CanAutoApproveDestructiveRewrite("claude", nil) {
		t.Fatal("reliable remote model should be eligible for risky rewrite review")
	}
	if !CanAutoApproveDestructiveRewrite("ollama", map[string]interface{}{"deterministic_edit": true}) {
		t.Fatal("deterministically validated rewrite should be eligible")
	}
	meta := map[string]interface{}{
		"channel":            "user-flow-scenarios",
		"editor_agent_trust": editorTrustAutoApply,
	}
	if !CanAutoApproveDestructiveRewrite("ollama", meta) {
		t.Fatal("ollama should auto-approve destructive rewrite on regression scenario channel with auto_apply trust")
	}
	meta["editor_agent_trust"] = "ask_before_edits"
	if CanAutoApproveDestructiveRewrite("ollama", meta) {
		t.Fatal("ollama must not auto-approve without auto_apply/yolo trust")
	}
}

func TestRegressionHarnessAllowsDestructiveAutoApply(t *testing.T) {
	t.Setenv("NJ_REGRESSION", "")
	if regressionHarnessAllowsDestructiveAutoApply(map[string]interface{}{"channel": "general"}) {
		t.Fatal("general channel should not allow destructive auto-apply")
	}
	if !regressionHarnessAllowsDestructiveAutoApply(map[string]interface{}{"channel": "user-flow-scenarios"}) {
		t.Fatal("user-flow-scenarios should allow destructive auto-apply")
	}
	if !regressionHarnessAllowsDestructiveAutoApply(map[string]interface{}{"channel": "collab-scenarios"}) {
		t.Fatal("-scenarios suffix channels should allow destructive auto-apply")
	}
	t.Setenv("NJ_REGRESSION", "1")
	if !regressionHarnessAllowsDestructiveAutoApply(nil) {
		t.Fatal("NJ_REGRESSION=1 should allow destructive auto-apply")
	}
}
