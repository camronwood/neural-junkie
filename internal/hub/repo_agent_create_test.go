package hub

import "testing"

func TestParseRepoAgentCreateArgs(t *testing.T) {
	tests := []struct {
		name      string
		parts     []string
		flagModel string
		wantName  string
		wantProv  string
		wantModel string
	}{
		{
			name:      "name then ollama with flag model",
			parts:     []string{"/create-repo-agent", "/tmp/app", "MyExpert", "ollama"},
			flagModel: "llama3.1",
			wantName:  "myexpert",
			wantProv:  "ollama",
			wantModel: "llama3.1",
		},
		{
			name:      "path only uses defaults",
			parts:     []string{"/create-repo-agent", "/tmp/app"},
			flagModel: "",
			wantName:  "",
			wantProv:  "ollama",
			wantModel: "",
		},
		{
			name:      "provider only no name",
			parts:     []string{"/create-repo-agent", "/tmp/app", "ollama", "qwen2.5-coder:14b"},
			flagModel: "",
			wantName:  "",
			wantProv:  "ollama",
			wantModel: "qwen2.5-coder:14b",
		},
		{
			name:      "multi word name before provider",
			parts:     []string{"/create-repo-agent", "/tmp/app", "My", "App", "Expert", "claude"},
			flagModel: "",
			wantName:  "my-app-expert",
			wantProv:  "claude",
			wantModel: "",
		},
		{
			name:      "legacy mistaken join avoided",
			parts:     []string{"/create-repo-agent", "/tmp/app", "neural-junkie-expert", "ollama"},
			flagModel: "",
			wantName:  "neural-junkie-expert",
			wantProv:  "ollama",
			wantModel: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotProv, gotModel := parseRepoAgentCreateArgs(tt.parts, tt.flagModel)
			if gotName != tt.wantName || gotProv != tt.wantProv || gotModel != tt.wantModel {
				t.Fatalf("parseRepoAgentCreateArgs() = (%q, %q, %q), want (%q, %q, %q)",
					gotName, gotProv, gotModel, tt.wantName, tt.wantProv, tt.wantModel)
			}
		})
	}
}
