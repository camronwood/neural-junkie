package routing

import "strings"

func normText(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func containsAny(text string, phrases ...string) bool {
	for _, p := range phrases {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}

func looksSecurity(text string) bool {
	return containsAny(text,
		"security", "auth", "oauth", "jwt", "encrypt", "crypt", "owasp",
		"penetration", "vulnerability", "cve", "compliance", "gdpr", "hipaa",
		"gosec", "npm audit", "scan_secrets", "gitleaks", "govulncheck",
		"secret scan",
	)
}

func looksCheap(text string) bool {
	return containsAny(text,
		"typo", "wording", "rephrase", "shorten", "grammar", "polish",
		"tweak", "rename", "comment", "whitespace", "formatting",
	)
}

func looksBiology(text string) bool {
	return containsAny(text,
		"biology", "protein", "dna", "rna", "genome", "enzyme", "cell",
		"sequence", "amino acid", "pcr", "crispr", "pathway", "organism",
		"biochemistry", "molecular", "esmfold", "fasta", "peptide",
		"analyze_sequence", "fold_protein", "fold this", "fold the",
	)
}

func looksFrontend(text string) bool {
	return containsAny(text,
		"frontend", "react", "vue", "css", "tailwind", "ui component",
		"accessibility", "a11y", "eslint", "typescript check", "run_typescript",
		"tsc ", "stylelint", "analyze_css",
	)
}

func looksBackend(text string) bool {
	return containsAny(text,
		"backend", "api endpoint", "database schema", "sql query", "rest api",
		"graphql", "analyze_go_code", "run_go_tests", "go test", "go vet",
		"staticcheck", "golangci-lint", "go mod", "gosec",
	)
}

func looksDevOps(text string) bool {
	return containsAny(text,
		"devops", "kubernetes", "docker", "terraform", "ci/cd", "deployment",
		"helm", "kubectl", "validate_yaml", "check_pod_logs", "prometheus",
	)
}

func looksArchitecture(text string) bool {
	return containsAny(text,
		"architecture", "system design", "microservice", "bounded context",
		"c4 model", "architecture review", "validate infrastructure",
	)
}

func looksCodeReview(text string) bool {
	return containsAny(text,
		"code review", "review this pr", "review the diff", "nitpick",
		"style guide", "run linter", "run analysis",
	)
}

func looksDatabase(text string) bool {
	return containsAny(text,
		"explain_query", "explain query", "check_indexes", "check indexes",
		"validate_schema", "validate schema", "generate_migration",
		"migration sql", "suggest_optimizations", "query optimization",
	)
}

func looksRust(text string) bool {
	return containsAny(text,
		"rust", "cargo clippy", "cargo test", "cargo audit",
		"check_cargo_toml", "rust lint",
	)
}

func looksCAD(text string) bool {
	return containsAny(text,
		"openscad", "cad", "3d model", "stl", "scad",
	)
}

func toolNeedForAgentType(agentType, text string) bool {
	text = normText(text)
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "biology":
		return containsAny(text,
			"analyze_sequence", "fold_protein", "esmfold", "fold this",
			"fold the", "sequence analysis", "reverse complement", "pdb",
			"analyze this sequence", "fold the sequence", "run fold",
			"structure prediction", "run_12plex_qc",
		) || (containsAny(text, "dna", "rna", "peptide") &&
			containsAny(text, "fold", "analyze", "sequence"))
	case "backend":
		return looksBackendTools(text)
	case "devops":
		return looksDevOpsTools(text)
	case "database":
		return looksDatabaseTools(text)
	case "frontend":
		return looksFrontendTools(text)
	case "security":
		return looksSecurityTools(text)
	case "code-review", "code_review":
		return looksCodeReviewTools(text)
	case "architecture":
		return looksArchitectureTools(text)
	case "rust":
		return looksRustTools(text)
	case "cad":
		return looksCAD(text)
	default:
		return false
	}
}

func looksBackendTools(q string) bool {
	return containsAny(q,
		"analyze_go_code", "run_go_tests", "go test", "go vet", "staticcheck",
		"golangci-lint", "check_dependencies", "go mod", "race condition",
		"analyze this go", "run tests on", "profile performance", "gosec",
	)
}

func looksDevOpsTools(q string) bool {
	return containsAny(q,
		"kubectl", "kubernetes", "validate_yaml", "validate yaml", "pod logs",
		"check_pod_logs", "prometheus", "docker image", "helm", "check_docker",
	)
}

func looksDatabaseTools(q string) bool {
	return containsAny(q,
		"explain_query", "explain query", "check_indexes", "check indexes",
		"validate_schema", "validate schema", "generate_migration", "migration sql",
		"suggest_optimizations", "query optimization", "table stats",
	)
}

func looksFrontendTools(q string) bool {
	return containsAny(q,
		"eslint", "typescript check", "run_typescript", "tsc ", "a11y",
		"accessibility audit", "check_package_json", "analyze_css", "stylelint",
	)
}

func looksSecurityTools(q string) bool {
	return containsAny(q,
		"gosec", "npm audit", "scan_secrets", "gitleaks", "govulncheck",
		"security headers", "cve", "vulnerability scan", "secret scan",
	)
}

func looksCodeReviewTools(q string) bool {
	return looksBackendTools(q) || looksFrontendTools(q) ||
		containsAny(q, "code review tool", "run linter", "run analysis")
}

func looksArchitectureTools(q string) bool {
	return looksDevOpsTools(q) || looksDatabaseTools(q) ||
		containsAny(q, "architecture review", "validate infrastructure", "schema review")
}

func looksRustTools(q string) bool {
	return containsAny(q,
		"cargo clippy", "cargo test", "cargo audit", "check_cargo_toml", "rust lint",
	)
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

func selectLoRATag(in Input) (tag, reason string) {
	text := normText(in.Text)
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

func detectDomain(text, agentType string) (domain, reason string) {
	text = normText(text)
	checks := []struct {
		fn     func(string) bool
		domain string
		reason string
	}{
		{looksSecurity, DomainSecurity, "domain_security"},
		{looksBiology, DomainBiology, "domain_biology"},
		{looksFrontend, DomainFrontend, "domain_frontend"},
		{looksBackend, DomainBackend, "domain_backend"},
		{looksDevOps, DomainDevOps, "domain_devops"},
		{looksArchitecture, DomainArchitecture, "domain_architecture"},
		{looksCodeReview, DomainCodeReview, "domain_code_review"},
		{looksDatabase, DomainDatabase, "domain_database"},
		{looksRust, DomainRust, "domain_rust"},
		{looksCAD, DomainCAD, "domain_cad"},
	}
	for _, c := range checks {
		if c.fn(text) {
			return c.domain, c.reason
		}
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if agentType != "" && agentType != "general" && agentType != "expert" && agentType != "moderator" && agentType != "assistant" {
		return agentType, "agent_type_default"
	}
	return DomainGeneral, "domain_general"
}

func detectCostTier(text string, hasImages bool) (tier, reason string) {
	if hasImages {
		return CostStandard, "vision_task"
	}
	text = normText(text)
	if looksCheap(text) && len(text) < 1200 {
		return CostCheap, "cheap_task"
	}
	if looksSecurity(text) {
		return CostPremium, "security_task"
	}
	return CostStandard, "standard_task"
}
