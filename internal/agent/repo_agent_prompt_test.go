package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

func TestBuildRepoPromptUsesSystemSeparatorAndRepoContext(t *testing.T) {
	ra := &RepoAgent{
		Agent: &Agent{
			Info: protocol.AgentInfo{Name: "test-expert"},
		},
		index: &repo.RepositoryIndex{
			Name:            "my-app",
			ArchitectureDoc: "ARCHITECTURE_MARKER: multi-agent hub",
			KeyFiles: map[string]string{
				"README.md": "# My App\nCore service.",
			},
			SourceFiles: map[string]*repo.SourceFile{
				"main.go": {
					Path:     "main.go",
					Language: "Go",
					Content:  mustCompressContent(t, "package main\nfunc main() {}"),
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "User"}, "What does this project do?")
	prompt := ra.buildRepoPrompt(msg, ra.index)

	system, user := ai.SplitSystemPrompt(prompt)
	if system == "" {
		t.Fatal("expected non-empty system prompt with repo context")
	}
	if user == "" {
		t.Fatal("expected non-empty user question section")
	}
	if !strings.Contains(system, "ARCHITECTURE_MARKER") {
		t.Fatal("architecture doc should be in system prompt")
	}
	if !strings.Contains(system, "README.md") {
		t.Fatal("key files should be in system prompt")
	}
	if !strings.Contains(user, "What does this project do?") {
		t.Fatal("user question should be after separator")
	}
}

func mustCompressContent(t *testing.T, content string) string {
	t.Helper()
	compressed, _, err := repo.CompressContent(content)
	if err != nil {
		t.Fatal(err)
	}
	return compressed
}
