package agent

import (
	"strings"
	"testing"
)

func TestValidateProposal_rejectsVueInReactRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.2.0"}}`)
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	m := DetectStackManifest(dir)

	err := ValidateProposal(dir, "src/App.vue", ProposalOpCreate, m)
	if err == nil || !strings.Contains(err.Error(), "vue") {
		t.Fatalf("expected vue rejection, got %v", err)
	}
}

func TestRedirectProposalPath_tailwind(t *testing.T) {
	t.Parallel()
	m := &StackManifest{TailwindConfig: "tailwind.config.js"}
	got := RedirectProposalPath("src-tauri/tailwind.config.js", m)
	if got != "tailwind.config.js" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateProposal_rejectsWrongTailwindPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "tailwind.config.js", "module.exports = {}\n")
	m := DetectStackManifest(dir)

	err := ValidateProposal(dir, "src-tauri/tailwind.config.js", ProposalOpCreate, m)
	if err == nil || !strings.Contains(err.Error(), "tailwind.config.js") {
		t.Fatalf("expected tailwind path rejection, got %v", err)
	}
}

func TestValidateProposal_editMustExist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	m := DetectStackManifest(dir)

	err := ValidateProposal(dir, "src/components/Missing.tsx", ProposalOpEdit, m)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing file error, got %v", err)
	}
}

func TestValidateProposal_allowsReactPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.2.0"}}`)
	writeFile(t, dir, "tailwind.config.js", "module.exports = {}\n")
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	m := DetectStackManifest(dir)

	if err := ValidateProposal(dir, "tailwind.config.js", ProposalOpEdit, m); err != nil {
		t.Fatalf("tailwind edit: %v", err)
	}
	if err := ValidateProposal(dir, "src/components/ThemeToggle.tsx", ProposalOpCreate, m); err != nil {
		t.Fatalf("new tsx component: %v", err)
	}
}
