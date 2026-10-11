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

func TestRedirectProposalPath_appJsDecoyWhenTsxOnDisk(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.2.0"}}`)
	writeFile(t, dir, "src/App.tsx", "export default function App() { return <main>ALPHA_MARKER</main>; }\n")
	writeFile(t, dir, "src/App.js", "export default function App() { return 'DECOY'; }\n")
	m := DetectStackManifest(dir)
	got := RedirectProposalPathForWorkspace("src/App.js", dir, m, "edit")
	if got != "src/App.tsx" {
		t.Fatalf("got %q want src/App.tsx", got)
	}
	err := ValidateProposal(dir, "src/App.js", ProposalOpEdit, m)
	if err == nil || !strings.Contains(err.Error(), "src/App.tsx") {
		t.Fatalf("expected decoy edit rejection, got %v", err)
	}
}

func TestDisambiguateAmbiguousBasename_prefersUserDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "pkg/alpha/config.go", "package alpha\n")
	writeFile(t, dir, "pkg/beta/config.go", "package beta\n")
	got, err := DisambiguateAmbiguousBasename(dir, "config.go", "please edit pkg/alpha/config.go only")
	if err != nil {
		t.Fatal(err)
	}
	if got != "pkg/alpha/config.go" {
		t.Fatalf("got %q", got)
	}
	_, err = DisambiguateAmbiguousBasename(dir, "config.go", "please update config.go")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
	// Explicit nested greenfield path must not be rejected when other basenames exist.
	got, err = DisambiguateAmbiguousBasename(dir, "pkg/gamma/config.go", "add pkg/gamma/config.go")
	if err != nil {
		t.Fatal(err)
	}
	if got != "pkg/gamma/config.go" {
		t.Fatalf("greenfield nested create got %q", got)
	}
}

func TestTargetGrounded_requiresExactRead(t *testing.T) {
	t.Parallel()
	st := &ImplementationSessionState{
		DiscoverTools: []string{"glob_file_search"},
		SeedsLoaded:   2,
	}
	if st.targetGrounded("src/App.tsx") {
		t.Fatal("glob/seeds count alone must not ground an unread target")
	}
	st.RecordReadPath("src/App.tsx")
	if !st.targetGrounded("src/App.tsx") {
		t.Fatal("exact read should ground target")
	}
	if st.targetGrounded("src/App.js") {
		t.Fatal("read of App.tsx must not ground App.js")
	}
}
