package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestDetectBareFilenames(t *testing.T) {
	t.Parallel()
	paths := DetectFilePaths("Emit [FILE_CHANGE] for tailwind.config.js and src/App.tsx")
	want := map[string]bool{"tailwind.config.js": true, "src/App.tsx": true}
	for p := range want {
		found := false
		for _, got := range paths {
			if got == p {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("DetectFilePaths missing %q in %v", p, paths)
		}
	}
}

func TestAppendImplementationSeedFiles_loadsTailwind(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := "module.exports = { darkMode: 'class' }\n"
	if err := os.WriteFile(dir+"/tailwind.config.js", []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	msg := &protocol.Message{Content: "please implement light/dark themes"}
	n := AppendImplementationSeedFiles(&b, nil, msg, dir, protocol.AgentTypeFrontend, nil)
	if n < 1 {
		t.Fatalf("expected at least tailwind loaded, got %d", n)
	}
	out := b.String()
	if !strings.Contains(out, "darkMode") {
		t.Fatalf("expected tailwind content in prompt, got %q", out)
	}
}
