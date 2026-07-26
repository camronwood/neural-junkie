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

func TestRedirectProposalPath_stripsScenarioBaseline(t *testing.T) {
	t.Parallel()
	got := RedirectProposalPath(".scenario-baseline/Makefile", nil)
	if got != "Makefile" {
		t.Fatalf("got %q want Makefile", got)
	}
	got = RedirectProposalPath(".scenario-baseline/src/App.tsx", &StackManifest{HasReact: true, EntryPoint: "src/App.tsx"})
	if got != "src/App.tsx" {
		t.Fatalf("got %q", got)
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

func TestRedirectProposalPath_appJsToTsx(t *testing.T) {
	t.Parallel()
	m := &StackManifest{HasReact: true, EntryPoint: "src/App.tsx"}
	got := RedirectProposalPath("src/App.js", m)
	if got != "src/App.tsx" {
		t.Fatalf("got %q", got)
	}
	// Deletes must keep App.js so boot-fix can remove the corrupt entry.
	got = RedirectProposalPathForOp("src/App.js", m, "delete")
	if got != "src/App.js" {
		t.Fatalf("delete redirect got %q want src/App.js", got)
	}
}

func TestValidateProposal_rejectsCreateAppJsWhenTsxEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.2.0"}}`)
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	m := DetectStackManifest(dir)

	err := ValidateProposal(dir, "src/App.js", ProposalOpCreate, m)
	if err == nil || !strings.Contains(err.Error(), "src/App.tsx") {
		t.Fatalf("expected entry conflict, got %v", err)
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
