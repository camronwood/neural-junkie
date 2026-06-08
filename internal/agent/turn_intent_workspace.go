package agent

import "strings"

// userAsksAboutWorkspaceVisibility reports whether the user is asking if the agent can see their project/files.
func userAsksAboutWorkspaceVisibility(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	// Project/workspace visibility — not single-file editor questions (those use editor review guidance).
	markers := []string{
		"see my workspace", "see the workspace", "see my project", "see my repo",
		"see my codebase", "workspace i have open", "my workspace",
		"have access to my workspace", "access to my workspace",
		"you have workspace access", "you have workspace", "have workspace access",
		"workspace access", "given you workspace", "granted workspace",
		"what files do you see", "files in my workspace",
		"what is in my workspace", "what's in my workspace",
	}
	if strings.Contains(lower, "workspace") || strings.Contains(lower, "codebase") ||
		strings.Contains(lower, "project") || strings.Contains(lower, "repo") {
		for _, m := range []string{"can you see", "do you see", "are you able to see"} {
			if strings.Contains(lower, m) {
				return true
			}
		}
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
