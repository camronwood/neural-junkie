package mcp

import "strings"

// hubMCPToolNames lists in-process hub MCP tools that must never be offered as shell commands.
var hubMCPToolNames = map[string]bool{
	// Biology
	"analyze_sequence":            true,
	"fold_protein":                true,
	"summarize_scan_summary":      true,
	"summarize_scan_analysis":     true,
	"run_12plex_qc":               true,
	"summarize_panel_qc":          true,
	"summarize_comparator_output": true,
	"run_secondary_analysis":      true,
	// CAD
	"write_openscad":       true,
	"render_openscad":      true,
	"list_openscad_params": true,
	"export_cad":           true,
	// Workspace
	"read_file":        true,
	"grep":             true,
	"glob_file_search": true,
	"list_dir":         true,
	"semantic_search":  true,
	"run_command":      true,
	// Repo
	"search_codebase":  true,
	"get_file_content": true,
	"search_by_path":   true,
	"list_key_files":   true,
	"git_log":          true,
	// Resources
	"list_exported_agents": true,
	"get_agent_resource":   true,
	"get_agent_prompt":     true,
	"recreate_agent":       true,
	"get_agent_info":       true,
	"search_agents":        true,
	// Confluence
	"search_space":       true,
	"get_page":           true,
	"search_by_label":    true,
	"list_recent_pages":  true,
	// Architecture / database / backend / frontend / security / rust / codereview / devops
	"validate_yaml":              true,
	"validate_schema":            true,
	"explain_query":              true,
	"check_dependencies":         true,
	"analyze_go_code":            true,
	"run_go_tests":               true,
	"run_eslint":                 true,
	"run_typescript_check":       true,
	"cargo_clippy":               true,
	"cargo_test":                 true,
	"cargo_audit":                true,
	"check_cargo_toml":           true,
	"run_gosec":                  true,
	"run_npm_audit":              true,
	"scan_secrets":               true,
	"check_go_vulnerabilities":   true,
	"validate_security_headers":  true,
	"check_package_json":         true,
	"analyze_css":                true,
	"check_indexes":              true,
	"suggest_optimizations":      true,
	"check_table_stats":          true,
	"generate_migration":         true,
	"kubectl_query":              true,
	"check_docker_image":         true,
	"check_pod_logs":             true,
	"query_prometheus":           true,
	"profile_performance":        true,
	"detect_race_conditions":     true,
	// AWS
	"aws_get_caller_identity": true,
	"aws_list_profiles":       true,
	"aws_sso_login_hint":      true,
	"aws_cli_query":           true,
	// Incident / Jira
	"jira_get_issue":       true,
	"jira_search_issues":   true,
	"jira_add_comment":     true,
	"jira_summarize_issue": true,
	// Built-in agent tools
	"generate_image": true,
}

// IsHubMCPToolName reports whether name is a hub MCP tool (not a shell binary).
func IsHubMCPToolName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	return hubMCPToolNames[name]
}
