package routing

import (
	"strings"
)

// LoRAInput is the context for selecting a composed Ollama LoRA tag.
type LoRAInput struct {
	TaskText      string
	AgentType     string
	AgentModel    string
	InstalledTags map[string]struct{}
}

// SelectComposedTag picks an nj-* Ollama tag for a collaboration task when available.
func SelectComposedTag(in LoRAInput) (tag string, reason string) {
	text := strings.ToLower(strings.TrimSpace(in.TaskText))

	type rule struct {
		match  func(string) bool
		tag    string
		reason string
	}
	rules := []rule{
		{looksSecurity, "nj-security:14b", "security_lora_tag"},
		{looksBiology, "nj-biology:8b", "biology_lora_tag"},
		{looksFrontend, "nj-frontend:14b", "frontend_lora_tag"},
		{looksBackend, "nj-backend:14b", "backend_lora_tag"},
		{looksDevOps, "nj-devops:14b", "devops_lora_tag"},
		{looksArchitecture, "nj-architecture:14b", "architecture_lora_tag"},
		{looksCodeReview, "nj-code-review:14b", "code_review_lora_tag"},
	}

	for _, r := range rules {
		if r.match(text) && tagInstalled(in.InstalledTags, r.tag) {
			return r.tag, r.reason
		}
	}

	agentModel := strings.TrimSpace(in.AgentModel)
	if isComposedNJTag(agentModel) && tagInstalled(in.InstalledTags, agentModel) {
		return agentModel, "agent_configured_lora"
	}

	agentType := strings.ToLower(strings.TrimSpace(in.AgentType))
	if agentType != "" {
		candidate := "nj-" + agentType + ":14b"
		if agentType == "biology" {
			candidate = "nj-biology:8b"
		}
		if tagInstalled(in.InstalledTags, candidate) {
			return candidate, "agent_type_lora"
		}
	}

	return "", "no_lora_override"
}

func tagInstalled(tags map[string]struct{}, tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" || tags == nil {
		return false
	}
	_, ok := tags[tag]
	return ok
}

func isComposedNJTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	return strings.HasPrefix(tag, "nj-") && strings.Contains(tag, ":")
}

func looksBiology(text string) bool {
	keywords := []string{
		"biology", "protein", "dna", "rna", "genome", "enzyme", "cell",
		"sequence", "amino acid", "pcr", "crispr", "pathway", "organism",
		"biochemistry", "molecular", "esmfold", "fasta",
	}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func looksFrontend(text string) bool {
	keywords := []string{"frontend", "react", "vue", "css", "tailwind", "ui component", "accessibility", "a11y"}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func looksBackend(text string) bool {
	keywords := []string{"backend", "api endpoint", "database schema", "sql query", "rest api", "graphql"}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func looksDevOps(text string) bool {
	keywords := []string{"devops", "kubernetes", "docker", "terraform", "ci/cd", "deployment", "helm"}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func looksArchitecture(text string) bool {
	keywords := []string{"architecture", "system design", "microservice", "bounded context", "c4 model"}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func looksCodeReview(text string) bool {
	keywords := []string{"code review", "review this pr", "review the diff", "nitpick", "style guide"}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}
