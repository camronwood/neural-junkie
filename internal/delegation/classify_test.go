package delegation

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClassifyForAgentDomainTools(t *testing.T) {
	cases := []struct {
		agentType protocol.AgentType
		question  string
		want      Intent
	}{
		{protocol.AgentTypeBackend, "please run go tests on ./cmd/server", IntentDomainTools},
		{protocol.AgentTypeBackend, "what is REST?", IntentDomainReasoning},
		{protocol.AgentTypeDevOps, "validate yaml for this deployment", IntentDomainTools},
		{protocol.AgentTypeDatabase, "explain query SELECT * FROM users", IntentDomainTools},
		{protocol.AgentTypeFrontend, "run eslint on src/", IntentDomainTools},
		{protocol.AgentTypeSecurity, "run gosec on the module", IntentDomainTools},
		{protocol.AgentTypeBiology, "fold this protein sequence", IntentDomainTools},
		{protocol.AgentTypeCodeReview, "run linter on this package", IntentDomainTools},
		{protocol.AgentTypeArchitecture, "validate schema for users table", IntentDomainTools},
		{protocol.AgentTypeRust, "cargo clippy on the crate", IntentDomainTools},
	}
	for _, tc := range cases {
		got := ClassifyForAgent(tc.agentType, tc.question)
		if got != tc.want {
			t.Fatalf("ClassifyForAgent(%s, %q) = %s, want %s", tc.agentType, tc.question, got, tc.want)
		}
	}
}
