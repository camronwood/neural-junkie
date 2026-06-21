package delegation

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RelevanceScore returns how well question matches agent expertise (higher = better fit).
func RelevanceScore(info protocol.AgentInfo, question string) int {
	content := strings.ToLower(strings.TrimSpace(question))
	if content == "" {
		return 0
	}
	words := strings.Fields(content)
	wordSet := make(map[string]bool, len(words))
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if len(word) >= 2 {
			wordSet[word] = true
		}
	}

	score := 0
	for _, skill := range info.Expertise {
		skillLower := strings.ToLower(skill)
		skillWords := strings.Fields(skillLower)
		for _, skillWord := range skillWords {
			skillWord = strings.Trim(skillWord, ".,!?;:")
			if len(skillWord) >= 2 && wordSet[skillWord] {
				score += 2
			}
		}
		if len(skillWords) > 1 {
			if strings.Contains(content, strings.Join(skillWords, " ")) {
				score += 3
			}
		}
	}
	for _, keyword := range typeKeywords(info.Type) {
		keyword = strings.Trim(keyword, ".,!?;:")
		if len(keyword) >= 2 && wordSet[keyword] {
			score++
		}
		if strings.Contains(content, keyword) {
			score++
		}
	}
	return score
}

func typeKeywords(t protocol.AgentType) []string {
	switch t {
	case protocol.AgentTypeFrontend:
		return []string{"ui", "frontend", "web", "desktop", "css", "html", "component", "user", "interface", "accessibility", "design"}
	case protocol.AgentTypeBackend:
		return []string{"api", "backend", "server", "endpoint", "service", "business", "logic", "integration", "cache", "queue"}
	case protocol.AgentTypeDevOps:
		return []string{"deploy", "deployment", "ci/cd", "docker", "kubernetes", "infrastructure", "monitoring",
			"aws", "azure", "gcp", "cloud", "terraform", "ansible", "pipeline", "ecs", "eks", "lambda"}
	case protocol.AgentTypeDatabase:
		return []string{"database", "sql", "query", "schema", "migration", "postgres", "mysql", "mongodb",
			"db", "documentdb", "dynamodb", "aurora", "rds", "nosql", "redis", "index"}
	case protocol.AgentTypeSecurity:
		return []string{"security", "auth", "authentication", "authorization", "encryption", "vulnerability", "xss",
			"iam", "ssl", "tls", "cors", "csrf", "rbac", "jwt", "oauth2", "secrets"}
	case protocol.AgentTypeRust:
		return []string{"rust", "cargo", "tokio", "ownership", "borrowing", "lifetime", "trait", "async", "unsafe", "wasm", "serde", "crate"}
	case protocol.AgentTypeArchitecture:
		return []string{"architecture", "architect", "system", "design", "scalability", "reliability", "tradeoff", "migration", "boundary", "integration"}
	case protocol.AgentTypeCodeReview:
		return []string{"review", "code", "correctness", "maintainability", "testing", "refactor", "regression", "readability", "quality"}
	case protocol.AgentTypeBiology:
		return []string{"biology", "protein", "gene", "genome", "dna", "rna", "sequence", "assay", "crispr",
			"enzyme", "mutation", "pathway", "cell", "lab", "protocol", "peptide", "amino", "fold", "esm"}
	case protocol.AgentTypeAWS:
		return []string{"aws", "sso", "iam", "ec2", "s3", "lambda", "cloudformation", "terraform",
			"vpc", "rds", "ecs", "eks", "cloudwatch", "route53", "dynamodb", "sqs", "sns"}
	case protocol.AgentTypeIncident:
		return []string{"incident", "bug", "ticket", "triage", "regression", "sentry", "jira",
			"stack", "trace", "severity", "outage", "postmortem", "reproduce"}
	case protocol.AgentTypeBrowser:
		return []string{"html", "css", "website", "webpage", "browser", "preview", "localhost",
			"dom", "responsive", "iframe", "fetch", "page", "viewport", "layout"}
	default:
		return nil
	}
}

// Resolve returns consultants sorted by score (desc) that beat the caller's self-score.
func Resolve(from protocol.AgentInfo, question string, candidates []protocol.AgentInfo, opts ResolveOptions) []Candidate {
	if opts.MinScore <= 0 {
		opts.MinScore = 2
	}
	if opts.MaxCandidates <= 0 {
		opts.MaxCandidates = 2
	}
	selfScore := RelevanceScore(from, question)
	var out []Candidate
	for _, c := range candidates {
		if c.ID == "" || c.ID == from.ID || c.ID == opts.ExcludeAgentID {
			continue
		}
		sc := RelevanceScore(c, question)
		if sc < opts.MinScore || sc <= selfScore {
			continue
		}
		out = append(out, Candidate{
			AgentID:   c.ID,
			AgentName: c.Name,
			AgentType: c.Type,
			Score:     sc,
			Intent:    ClassifyForAgent(c.Type, question),
		})
	}
	sortCandidates(out)
	if len(out) > opts.MaxCandidates {
		out = out[:opts.MaxCandidates]
	}
	return out
}

func sortCandidates(c []Candidate) {
	for i := 0; i < len(c); i++ {
		for j := i + 1; j < len(c); j++ {
			if c[j].Score > c[i].Score {
				c[i], c[j] = c[j], c[i]
			}
		}
	}
}
