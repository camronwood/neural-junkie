package protocol

import (
	"strings"
)

var stackToolPrefixes = []string{
	"docker", "docker-compose", "compose", "npm", "yarn", "pnpm", "npx",
	"kubectl", "helm", "terraform", "terragrunt", "aws", "gcloud", "az ",
	"make", "mvn", "gradle", "cargo run", "cargo build", "go run", "go build",
	"pip install", "poetry run", "bundle exec", "mix phx", "dotnet run",
}

var readOnlyInspectionPrefixes = []string{
	"ls", "pwd", "cd ", "cat ", "head ", "tail ", "grep ", "find ", "file ",
	"tree", "wc ", "sort ", "uniq ", "which ", "whereis ", "git status",
	"git log", "git diff", "git show", "git branch",
}

// LooksLikeStackToolCommand reports build/deploy/runtime tooling agents often
// suggest after seeing docker-compose.yml or package.json in a repo outline.
func LooksLikeStackToolCommand(command string) bool {
	command = strings.TrimSpace(strings.ToLower(command))
	if command == "" {
		return false
	}
	first := strings.Fields(command)
	if len(first) == 0 {
		return false
	}
	head := first[0]
	for _, prefix := range stackToolPrefixes {
		p := strings.Fields(prefix)
		if len(p) == 1 {
			if head == p[0] || strings.HasPrefix(head, p[0]) {
				return true
			}
			continue
		}
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

// LooksLikeReadOnlyInspectionCommand reports low-risk read/inspect shell usage.
func LooksLikeReadOnlyInspectionCommand(command string) bool {
	command = strings.TrimSpace(strings.ToLower(command))
	if command == "" {
		return false
	}
	for _, prefix := range readOnlyInspectionPrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}
