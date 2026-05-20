package collaboration

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// AssignSuggestion is an auto-assign recommendation for a task.
type AssignSuggestion struct {
	AgentID   string  `json:"agent_id"`
	AgentName string  `json:"agent_name"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason"`
}

const minSuggestScore = 2.0

var typeKeywords = map[protocol.AgentType][]string{
	protocol.AgentTypeSecurity: {"security", "auth", "oauth", "jwt", "encrypt", "crypt", "owasp", "vulnerability", "cve"},
	protocol.AgentTypeRust:     {"rust", "cargo", "wasm"},
	protocol.AgentTypeBiology:  {"biology", "protein", "gene", "genome", "dna", "rna", "sequence", "assay", "crispr", "mutation", "enzyme", "pathway", "lab", "protocol"},
	protocol.AgentTypeBackend:  {"go", "golang", "api", "rest", "graphql", "grpc", "backend", "server"},
	protocol.AgentTypeFrontend: {"react", "vue", "angular", "css", "ui", "ux", "frontend", "component"},
	protocol.AgentTypeDevOps:   {"docker", "kubernetes", "k8s", "ci", "cd", "terraform", "aws", "deploy", "devops"},
	protocol.AgentTypeDatabase: {"sql", "postgres", "mysql", "mongo", "database", "schema", "migration"},
}

// SuggestAssignee picks the best agent from pool for a task description.
func SuggestAssignee(pool []CollaborationAgent, title, description string, inFlight map[string]int) *AssignSuggestion {
	if len(pool) == 0 {
		return nil
	}
	text := strings.ToLower(title + " " + description)
	words := suggestWordSet(text)

	var best *AssignSuggestion
	for _, ag := range pool {
		score := scoreAgentForTask(ag, text, words)
		if inFlight != nil {
			score -= float64(inFlight[ag.AgentID]) * 0.25
		}
		if best == nil || score > best.Score {
			best = &AssignSuggestion{
				AgentID:   ag.AgentID,
				AgentName: ag.AgentName,
				Score:     score,
				Reason:    assignReasonFor(ag, score),
			}
		}
	}
	if best == nil || best.Score < minSuggestScore {
		return nil
	}
	return best
}

func scoreAgentForTask(ag CollaborationAgent, text string, words map[string]bool) float64 {
	var score float64
	if kws, ok := typeKeywords[ag.AgentType]; ok {
		for _, k := range kws {
			if strings.Contains(text, k) {
				score += 2
			}
		}
	}
	for _, skill := range ag.Expertise {
		for _, w := range strings.Fields(strings.ToLower(skill)) {
			w = strings.Trim(w, ".,!?;:")
			if len(w) >= 2 && words[w] {
				score += 1.5
			}
		}
		if strings.Contains(text, strings.ToLower(skill)) {
			score += 1
		}
	}
	return score
}

func assignReasonFor(ag CollaborationAgent, score float64) string {
	if score >= 4 {
		return "strong_expertise_match"
	}
	if score >= minSuggestScore {
		return "expertise_match"
	}
	return "low_confidence"
}

func suggestWordSet(text string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(text) {
		w = strings.Trim(w, ".,!?;:")
		if len(w) >= 2 {
			set[w] = true
		}
	}
	return set
}
