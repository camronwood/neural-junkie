package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	destructiveRewriteMinLines = 40
	destructiveRewriteRatio    = 0.50
)

type implementationFileSnapshot struct {
	content      []byte
	mode         os.FileMode
	existed      bool
	expectedHash [sha256.Size]byte
}

// IsDestructiveFileRewrite reports near-total replacements of established files.
// Common leading/trailing lines are treated as preserved to keep surgical edits cheap.
func IsDestructiveFileRewrite(oldContent, newContent string) (bool, float64) {
	oldLines := splitSafetyLines(oldContent)
	newLines := splitSafetyLines(newContent)
	if len(oldLines) < destructiveRewriteMinLines || len(newLines) == 0 {
		return false, 0
	}
	prefix := 0
	for prefix < len(oldLines) && prefix < len(newLines) && oldLines[prefix] == newLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(oldLines)-prefix && suffix < len(newLines)-prefix &&
		oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}
	preserved := prefix + suffix
	ratio := float64(len(oldLines)-preserved) / float64(len(oldLines))
	return ratio > destructiveRewriteRatio, ratio
}

func splitSafetyLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

func gitHeadFileContent(ctx context.Context, wsPath, path string) (string, bool) {
	wsPath = strings.TrimSpace(wsPath)
	path = normalizeFileChangeRelPath(path)
	if wsPath == "" || !isValidFileChangeRelPath(path) {
		return "", false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	gitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(gitCtx, "git", "-C", wsPath, "show", "HEAD:"+filepath.ToSlash(path)).Output()
	if err != nil || len(out) > 1024*1024 {
		return "", false
	}
	return string(out), true
}

func gitBaselineRewriteRisk(ctx context.Context, wsPath, path, newContent string) (bool, float64, int) {
	baseline, ok := gitHeadFileContent(ctx, wsPath, path)
	if !ok {
		return false, 0, 0
	}
	risky, ratio := IsDestructiveFileRewrite(baseline, newContent)
	return risky, ratio, len(splitSafetyLines(baseline))
}

func normalizedProposalContent(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func proposalContentHash(path, content string) [sha256.Size]byte {
	return sha256.Sum256([]byte(normalizeFileChangeRelPath(path) + "\x00" + normalizedProposalContent(content)))
}

func (s *ImplementationSessionState) prepareEditSnapshot(wsPath, path, oldContent, newContent string) error {
	if s == nil {
		return nil
	}
	if normalizedProposalContent(oldContent) == normalizedProposalContent(newContent) {
		s.CircuitBreakerFired = true
		return fmt.Errorf("duplicate/no-op edit blocked for %s: resulting content is unchanged", path)
	}
	hash := proposalContentHash(path, newContent)
	if s.ProposalContentHashes == nil {
		s.ProposalContentHashes = make(map[[sha256.Size]byte]struct{})
	}
	if _, exists := s.ProposalContentHashes[hash]; exists {
		s.CircuitBreakerFired = true
		return fmt.Errorf("duplicate edit blocked for %s: this exact resulting content was already proposed", path)
	}
	if strings.TrimSpace(wsPath) == "" {
		return nil
	}
	if s.FileSnapshots == nil {
		s.FileSnapshots = make(map[string]*implementationFileSnapshot)
	}
	path = normalizeFileChangeRelPath(path)
	if _, exists := s.FileSnapshots[path]; exists {
		return nil
	}
	abs := filepath.Join(wsPath, path)
	snap := &implementationFileSnapshot{}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		body, readErr := os.ReadFile(abs)
		if readErr != nil {
			return fmt.Errorf("snapshot %s: %w", path, readErr)
		}
		snap.content = body
		snap.mode = info.Mode().Perm()
		snap.existed = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s: %w", path, err)
	}
	s.FileSnapshots[path] = snap
	return nil
}

func (s *ImplementationSessionState) recordEditResult(path, newContent string) {
	if s == nil {
		return
	}
	hash := proposalContentHash(path, newContent)
	if s.ProposalContentHashes == nil {
		s.ProposalContentHashes = make(map[[sha256.Size]byte]struct{})
	}
	s.ProposalContentHashes[hash] = struct{}{}
	if snap := s.FileSnapshots[normalizeFileChangeRelPath(path)]; snap != nil {
		snap.expectedHash = sha256.Sum256([]byte(normalizedProposalContent(newContent)))
	}
}

// releaseSnapshot drops the rollback snapshot for path so a successful direct-apply
// (e.g. Makefile playbook under auto_apply) is not restored on session exit.
func (s *ImplementationSessionState) releaseSnapshot(path string) {
	if s == nil || s.FileSnapshots == nil {
		return
	}
	delete(s.FileSnapshots, normalizeFileChangeRelPath(path))
}

func (s *ImplementationSessionState) shouldRollback() bool {
	if s == nil || s.TrustMode != editorTrustAutoApply || len(s.FileSnapshots) == 0 {
		return false
	}
	return s.CircuitBreakerFired || (s.VerifyFailed && !s.VerifySkipped) || s.ConsecutiveNoVerifyProgress > 0
}

// rollbackFailedAutoApplySession restores only files whose current contents still
// match this session's last proposal, avoiding clobbering concurrent user edits.
func (s *ImplementationSessionState) rollbackFailedAutoApplySession(wsPath string) {
	if !s.shouldRollback() || strings.TrimSpace(wsPath) == "" {
		return
	}
	for path, snap := range s.FileSnapshots {
		abs := filepath.Join(wsPath, path)
		current, err := os.ReadFile(abs)
		if err != nil && !os.IsNotExist(err) {
			s.RollbackErrors = append(s.RollbackErrors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if sha256.Sum256([]byte(normalizedProposalContent(string(current)))) != snap.expectedHash {
			continue
		}
		if snap.existed {
			mode := snap.mode
			if mode == 0 {
				mode = 0o644
			}
			if err := os.WriteFile(abs, snap.content, mode); err != nil {
				s.RollbackErrors = append(s.RollbackErrors, fmt.Sprintf("%s: %v", path, err))
				continue
			}
		} else if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
			s.RollbackErrors = append(s.RollbackErrors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		s.RolledBackFiles = appendUnique(s.RolledBackFiles, []string{path})
	}
}

func (s *ImplementationSessionState) recordVerificationProgress(output string, failed, skipped bool) {
	if s == nil || skipped {
		return
	}
	signature := sha256.Sum256([]byte(strings.TrimSpace(output)))
	score := verificationFailureScore(output, failed)
	if s.VerificationRuns > 0 && failed {
		improved := s.LastVerifyFailed && score < s.LastVerifyFailureScore
		if !improved {
			s.ConsecutiveNoVerifyProgress++
			s.CircuitBreakerFired = true
		} else {
			s.ConsecutiveNoVerifyProgress = 0
		}
	} else if !failed {
		s.ConsecutiveNoVerifyProgress = 0
	}
	s.VerificationRuns++
	s.LastVerifySignature = signature
	s.LastVerifyFailureScore = score
	s.LastVerifyFailed = failed
}

func verificationFailureScore(output string, failed bool) int {
	if !failed {
		return 0
	}
	score := 0
	for _, line := range splitSafetyLines(strings.ToLower(output)) {
		if strings.Contains(line, "error") || strings.Contains(line, "failed") || strings.Contains(line, "failure") {
			score++
		}
	}
	if score == 0 {
		return 1
	}
	return score
}

// CanAutoApproveDestructiveRewrite implements model trust tiers for risky rewrites.
// Local models leave the proposal pending for human approval; reliable remote models
// and deterministic validators may auto-approve.
func CanAutoApproveDestructiveRewrite(msgProvider string, metadata map[string]interface{}) bool {
	if metadata != nil {
		if approved, _ := metadata["deterministic_edit"].(bool); approved {
			return true
		}
		if approved, _ := metadata["large_rewrite_approved"].(bool); approved {
			return true
		}
	}
	switch strings.ToLower(strings.TrimSpace(msgProvider)) {
	case "claude", "anthropic", "openai", "cursor":
		return true
	default:
		return false
	}
}
