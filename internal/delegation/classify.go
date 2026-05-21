package delegation

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ClassifyForAgent picks consult intent for a target specialist type.
func ClassifyForAgent(agentType protocol.AgentType, question string) Intent {
	q := strings.ToLower(question)
	if agentType == protocol.AgentTypeBiology {
		if looksBioTools(q) {
			return IntentDomainTools
		}
		return IntentDomainReasoning
	}
	return IntentDomainReasoning
}

func looksBioTools(q string) bool {
	tools := []string{
		"analyze_sequence", "fold_protein", "esmfold", "fold this", "fold the",
		"sequence analysis", "reverse complement", "pdb", "analyze this sequence",
		"fold the sequence", "run fold", "structure prediction",
	}
	for _, t := range tools {
		if strings.Contains(q, t) {
			return true
		}
	}
	if strings.Contains(q, "dna") || strings.Contains(q, "rna") || strings.Contains(q, "peptide") {
		if strings.Contains(q, "fold") || strings.Contains(q, "analyze") || strings.Contains(q, "sequence") {
			return true
		}
	}
	return false
}

// ClassifyMessage inspects the user message for multi-domain signals.
func ClassifyMessage(_ protocol.AgentType, _ string, candidates []Candidate) Intent {
	if len(candidates) > 1 {
		return IntentMultiDomain
	}
	if len(candidates) == 1 {
		return candidates[0].Intent
	}
	return IntentGeneral
}
