package delegation

import "strings"

func containsAny(q string, phrases ...string) bool {
	for _, p := range phrases {
		if strings.Contains(q, p) {
			return true
		}
	}
	return false
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
