package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (s *ImplementationSessionState) recordRegistrationSuccess(path string) {
	if s == nil {
		return
	}
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return
	}
	s.RegisteredFiles = appendUnique(s.RegisteredFiles, []string{path})
}

func (s *ImplementationSessionState) recordRegistrationFailure(path string, err error) {
	if s == nil || err == nil {
		return
	}
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		path = "(unknown)"
	}
	line := fmt.Sprintf("%s: %s", path, strings.TrimSpace(err.Error()))
	for _, existing := range s.RegistrationErrors {
		if existing == line {
			return
		}
	}
	s.RegistrationErrors = append(s.RegistrationErrors, line)
}

func (s *ImplementationSessionState) hasRegisteredProposals() bool {
	return s != nil && len(s.RegisteredFiles) > 0
}

func shouldAppendFileChangeApprovalPrompt(msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	trust := strings.TrimSpace(msg.EditorAgentTrust())
	if trust == editorTrustAutoApply || trust == "yolo" {
		return false
	}
	return true
}

func (a *Agent) noteProposalResult(ctx context.Context, path string, err error) {
	st := implementationSessionStateFromContext(ctx)
	if st == nil {
		return
	}
	if err != nil {
		st.recordRegistrationFailure(path, err)
		return
	}
	st.recordRegistrationSuccess(path)
}
