package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	implCommandRepeatBlockThreshold = 2
	implCommandThrashingLimit       = 3
	commandBroadcastDedupeWindow    = 60 * time.Second
)

// CommandRunRecord tracks a single run_command invocation in an implementation session.
type CommandRunRecord struct {
	Cmd               string
	ExitCode          int
	StderrFingerprint string
	Round             int
	AfterReadOrEdit   bool
}

var commandBroadcastDedupe sync.Map

func normalizeImplCommand(cmd string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(cmd)), " ")
}

func fingerprintCommandOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	if len(output) > 200 {
		output = output[:200]
	}
	output = strings.Join(strings.Fields(output), " ")
	sum := sha256.Sum256([]byte(output))
	return hex.EncodeToString(sum[:8])
}

func (s *ImplementationSessionState) BootFixGatingEnabled() bool {
	return s != nil && s.BootFixIntent
}

func (s *ImplementationSessionState) BootFixReadsSatisfied() bool {
	if s == nil || !s.BootFixIntent {
		return true
	}
	for _, p := range s.BootReadPaths {
		if shared.CommandMatchesRequiredBootRead(p) {
			return true
		}
	}
	return false
}

func (s *ImplementationSessionState) RecordReadPath(path string) {
	if s == nil {
		return
	}
	path = shared.NormalizeCommandPath(path)
	if path == "" {
		return
	}
	for _, existing := range s.BootReadPaths {
		if existing == path {
			return
		}
	}
	s.BootReadPaths = append(s.BootReadPaths, path)
	s.LastReadPaths = appendUnique(s.LastReadPaths, []string{path})
	s.SinceLastCommandReadOrEdit = true
	s.clearCommandFailuresSinceEdit()
}

func (s *ImplementationSessionState) RecordEdit(path string) {
	if s == nil {
		return
	}
	path = shared.NormalizeCommandPath(path)
	if path != "" {
		s.LastReadPaths = appendUnique(s.LastReadPaths, []string{path})
	}
	s.SinceLastCommandReadOrEdit = true
	s.CommandOnlyRounds = 0
	s.clearCommandFailuresSinceEdit()
}

func (s *ImplementationSessionState) clearCommandFailuresSinceEdit() {
	if s == nil || s.CommandFailuresSinceEdit == nil {
		return
	}
	for k := range s.CommandFailuresSinceEdit {
		delete(s.CommandFailuresSinceEdit, k)
	}
}

func (s *ImplementationSessionState) bumpCommandFailure(cmd string) {
	if s == nil {
		return
	}
	cmd = normalizeImplCommand(cmd)
	if cmd == "" {
		return
	}
	if s.CommandFailuresSinceEdit == nil {
		s.CommandFailuresSinceEdit = make(map[string]int)
	}
	s.CommandFailuresSinceEdit[cmd]++
}

func (s *ImplementationSessionState) repeatedFailureCount(cmd string) int {
	if s == nil || s.CommandFailuresSinceEdit == nil {
		return 0
	}
	return s.CommandFailuresSinceEdit[normalizeImplCommand(cmd)]
}

func (s *ImplementationSessionState) RecordCommandRun(cmd string, exitCode int, output string) {
	if s == nil {
		return
	}
	cmd = normalizeImplCommand(cmd)
	if cmd == "" {
		return
	}
	fp := fingerprintCommandOutput(output)
	rec := CommandRunRecord{
		Cmd:               cmd,
		ExitCode:          exitCode,
		StderrFingerprint: fp,
		Round:             s.EditRound,
		AfterReadOrEdit:   s.SinceLastCommandReadOrEdit,
	}
	s.CommandHistory = append(s.CommandHistory, rec)
	s.LastCommandOutputText = output
	if exitCode != 0 {
		s.LastFailedCommand = cmd
		s.bumpCommandFailure(cmd)
	}
	if s.SinceLastCommandReadOrEdit {
		s.SinceLastCommandReadOrEdit = false
	}
}

func (s *ImplementationSessionState) ShouldBlockRunCommand(cmd string) error {
	if s == nil {
		return nil
	}
	cmd = normalizeImplCommand(cmd)
	if cmd == "" {
		return nil
	}
	if s.BootFixGatingEnabled() && shared.BootFixBootCommand(cmd) && !s.BootFixReadsSatisfied() {
		return shared.BootFixReadGateError()
	}
	if s.repeatedFailureCount(cmd) >= implCommandRepeatBlockThreshold {
		s.CircuitBreakerFired = true
		return shared.RepeatedCommandFailureError(cmd)
	}
	return nil
}

func (s *ImplementationSessionState) LastCommandOutput() string {
	if s == nil {
		return ""
	}
	return s.LastCommandOutputText
}

func (s *ImplementationSessionState) CircuitBreakerTriggered() bool {
	return s != nil && s.CircuitBreakerFired
}

func (s *ImplementationSessionState) PlaybookUsed() string {
	if s == nil {
		return ""
	}
	return s.PlaybookUsedName
}

func (s *ImplementationSessionState) SetPlaybookUsed(name string) {
	if s == nil {
		return
	}
	s.PlaybookUsedName = strings.TrimSpace(name)
}

func (s *ImplementationSessionState) CommandFailureSummary() []shared.CommandFailureCount {
	if s == nil {
		return nil
	}
	counts := make(map[string]int)
	for _, rec := range s.CommandHistory {
		if rec.ExitCode == 0 {
			continue
		}
		counts[rec.Cmd]++
	}
	if len(counts) == 0 {
		return nil
	}
	out := make([]shared.CommandFailureCount, 0, len(counts))
	for cmd, n := range counts {
		out = append(out, shared.CommandFailureCount{Command: cmd, Count: n})
	}
	return out
}

func (s *ImplementationSessionState) commandThrashingDetected() bool {
	if s == nil {
		return false
	}
	if s.CommandOnlyRounds >= 2 {
		return true
	}
	for cmd, n := range s.failedCommandCounts() {
		if n >= implCommandThrashingLimit && s.repeatedFailureCount(cmd) >= implCommandThrashingLimit {
			return true
		}
	}
	return false
}

func (s *ImplementationSessionState) failedCommandCounts() map[string]int {
	out := make(map[string]int)
	for _, rec := range s.CommandHistory {
		if rec.ExitCode != 0 {
			out[rec.Cmd]++
		}
	}
	return out
}

func (s *ImplementationSessionState) formatCommandFailureSummaryLine() string {
	counts := s.failedCommandCounts()
	if len(counts) == 0 {
		return ""
	}
	var parts []string
	for cmd, n := range counts {
		if n <= 1 {
			continue
		}
		parts = append(parts, cmd+" (×"+itoa(n)+")")
	}
	if len(parts) == 0 {
		return ""
	}
	return "Repeated command failures: " + strings.Join(parts, ", ") + ". Read Makefile and add missing targets before re-running."
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func (a *Agent) observeImplementationSessionToolStep(
	roundCtx context.Context,
	msg *protocol.Message,
	state *ImplementationSessionState,
	streamMsgID string,
	ev ai.ToolStepEvent,
) {
	if state == nil {
		return
	}
	if ev.Kind == "result" {
		state.noteToolStep()
	}
	if ev.Kind == "result" && isDiscoverTool(ev.Name) {
		state.recordDiscoverTool(ev.Name)
	}
	if ev.Kind == "result" && (ev.Name == "propose_file_edit" || ev.Name == "search_replace" || ev.Name == "apply_patch") {
		state.RecordEdit("")
	}
	if streamMsgID != "" && a != nil && msg != nil {
		a.broadcastToolStep(roundCtx, msg, streamMsgID, ev)
	}
}

func attachImplSessionCommandPolicy(ctx context.Context, state *ImplementationSessionState) context.Context {
	if state == nil {
		return ctx
	}
	return shared.ContextWithCommandPolicy(ctx, state)
}

func shouldSkipDuplicateCommandBroadcast(channel, agentID, cmd, mcpResult string) bool {
	exitCode, _, _ := parseRunCommandMCPResult(mcpResult)
	fp := fingerprintCommandOutput(mcpResult)
	key := channel + "|" + agentID + "|" + normalizeImplCommand(cmd) + "|" + fp + "|" + strconvItoa(exitCode)
	now := time.Now()
	if prev, ok := commandBroadcastDedupe.Load(key); ok {
		if t, ok := prev.(time.Time); ok && now.Sub(t) < commandBroadcastDedupeWindow {
			return true
		}
	}
	commandBroadcastDedupe.Store(key, now)
	return false
}

func strconvItoa(n int) string {
	return itoa(n)
}

func (s *ImplementationSessionState) noteCommandOnlyRound(proposed bool) {
	if s == nil || proposed {
		return
	}
	if len(s.CommandHistory) == 0 {
		return
	}
	s.CommandOnlyRounds++
}

func commandOutputMatchesPlaybook(output string) string {
	output = strings.ToLower(output)
	switch {
	case strings.Contains(output, "no rule to make target 'start-all'"),
		strings.Contains(output, `no rule to make target "start-all"`),
		strings.Contains(output, "no rule to make target `start-all'"):
		return "missing_start_all_target"
	case strings.Contains(output, "devpath") && strings.Contains(output, "port"),
		strings.Contains(output, "connection refused") && strings.Contains(output, "1420"):
		return "tauri_vite_port_mismatch"
	default:
		return ""
	}
}
