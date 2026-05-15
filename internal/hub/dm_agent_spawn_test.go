package hub

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNormalizeExpertSlug(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"guitar,", "guitar"},
		{"  Guitar  ", "guitar"},
		{"legal-advice;", "legal-advice"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := normalizeExpertSlug(tc.in); got != tc.want {
			t.Errorf("normalizeExpertSlug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveExpertPreset(t *testing.T) {
	spec, err := ResolveExpert("rust", "")
	if err != nil {
		t.Fatal(err)
	}
	if !spec.IsPreset || spec.AgentType != protocol.AgentTypeRust {
		t.Fatalf("got %+v", spec)
	}
}

func TestResolveExpertCustom(t *testing.T) {
	spec, err := ResolveExpert("guitar", "Focus on jazz chords")
	if err != nil {
		t.Fatal(err)
	}
	if spec.IsPreset {
		t.Fatal("expected custom expert")
	}
	if spec.AgentType != protocol.AgentTypeHelper {
		t.Fatalf("type = %q", spec.AgentType)
	}
	if spec.Label != "Guitar" {
		t.Fatalf("label = %q", spec.Label)
	}
	if spec.PersonaMarkdown == "" {
		t.Fatal("expected persona markdown")
	}
	if !strings.Contains(spec.PersonaMarkdown, "jazz chords") {
		t.Fatalf("persona missing extra instructions: %q", spec.PersonaMarkdown)
	}
}

func TestParseCreateExpertParts(t *testing.T) {
	parts := parseCreateExpertParts([]string{"/create-expert", "guitar,", "GuitarCoach,", "ollama"})
	if len(parts) != 4 {
		t.Fatalf("len = %d, parts = %v", len(parts), parts)
	}
	if parts[1] != "guitar" || parts[2] != "GuitarCoach" || parts[3] != "ollama" {
		t.Fatalf("parts = %v", parts)
	}
}

func TestSplitCreateExpertArgs(t *testing.T) {
	tests := []struct {
		parts              []string
		slug, name, prov, model string
	}{
		{
			[]string{"/create-expert", "guitar", "GuitarCoach"},
			"guitar", "GuitarCoach", "", "",
		},
		{
			[]string{"/create-expert", "tabs", "Tabs"},
			"tabs", "Tabs", "", "",
		},
		{
			[]string{"/create-expert", "music", "Music", "Muisc"},
			"music", "Music Muisc", "", "",
		},
		{
			[]string{"/create-expert", "guitar", "GuitarCoach", "ollama"},
			"guitar", "GuitarCoach", "ollama", "",
		},
		{
			[]string{"/create-expert", "rust", "RustGuru", "ollama", "qwen2.5-coder:14b"},
			"rust", "RustGuru", "ollama", "qwen2.5-coder:14b",
		},
		{
			[]string{"/create-expert", "rust", "ollama"},
			"rust", "", "ollama", "",
		},
	}
	for _, tc := range tests {
		slug, name, prov, model := splitCreateExpertArgs(tc.parts)
		if slug != tc.slug || name != tc.name || prov != tc.prov || model != tc.model {
			t.Errorf("splitCreateExpertArgs(%v) = %q,%q,%q,%q want %q,%q,%q,%q",
				tc.parts, slug, name, prov, model, tc.slug, tc.name, tc.prov, tc.model)
		}
	}
}

func TestIsKnownExpertProvider(t *testing.T) {
	if !isKnownExpertProvider("ollama") || !isKnownExpertProvider("") {
		t.Fatal("expected known")
	}
	if isKnownExpertProvider("guitar,") || isKnownExpertProvider("tabs,") {
		t.Fatal("expected unknown")
	}
}

func TestExpertSlugToAgentTypeCustom(t *testing.T) {
	typ, err := ExpertSlugToAgentType("guitar")
	if err != nil {
		t.Fatal(err)
	}
	if typ != protocol.AgentTypeHelper {
		t.Fatalf("type = %q", typ)
	}
}
