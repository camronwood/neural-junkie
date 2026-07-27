package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	rustUnresolvedImportRE = regexp.MustCompile("(?i)error\\[e0432\\][^\\n]*`([a-z][a-z0-9_-]*)`")
	rustUndeclaredCrateRE  = regexp.MustCompile("(?i)error\\[e0433\\][^\\n]*`([a-z][a-z0-9_-]*)`")
	rustCannotFindCrateRE  = regexp.MustCompile("(?i)(?:cannot find crate|could not find) `([a-z][a-z0-9_-]*)`")
	invalidRustCrateNameRE = regexp.MustCompile(`[^a-z0-9_-]+`)
)

var rustStdPseudoCrates = map[string]bool{
	"std": true, "core": true, "alloc": true, "test": true, "proc_macro": true,
}

func extractMissingRustCrates(output string) []string {
	seen := make(map[string]bool)
	var crates []string
	add := func(crate string) {
		crate = strings.TrimSpace(crate)
		if crate == "" || rustStdPseudoCrates[crate] {
			return
		}
		if seen[crate] {
			return
		}
		seen[crate] = true
		crates = append(crates, crate)
	}
	for _, re := range []*regexp.Regexp{rustUnresolvedImportRE, rustUndeclaredCrateRE, rustCannotFindCrateRE} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return crates
}

func defaultRustCrateVersion(crate string) string {
	switch crate {
	case "rand":
		return "0.8"
	case "serde", "serde_json":
		return "1.0"
	case "tokio":
		return "1"
	default:
		return "1"
	}
}

func cargoTomlHasDependency(content, crate string) bool {
	crate = strings.TrimSpace(crate)
	if crate == "" {
		return false
	}
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(crate) + `\s*=`)
	return re.MatchString(content)
}

func addMissingDependencyToCargoToml(existing []byte, crate string) (string, bool) {
	crate = strings.TrimSpace(crate)
	if crate == "" {
		return "", false
	}
	content := string(existing)
	if cargoTomlHasDependency(content, crate) {
		return "", false
	}
	depLine := crate + ` = "` + defaultRustCrateVersion(crate) + `"` + "\n"
	if idx := strings.Index(content, "[dependencies]"); idx >= 0 {
		insertAt := idx + len("[dependencies]")
		rest := content[insertAt:]
		switch {
		case strings.HasPrefix(rest, "\r\n"):
			insertAt += 2
		case strings.HasPrefix(rest, "\n"):
			insertAt += 1
		}
		prefix := content[:insertAt]
		suffix := content[insertAt:]
		if strings.TrimSpace(suffix) != "" && !strings.HasPrefix(suffix, "\n") && !strings.HasPrefix(suffix, "\r\n") {
			depLine = "\n" + depLine
		} else if !strings.HasSuffix(prefix, "\n") && !strings.HasSuffix(prefix, "\r\n") {
			depLine = "\n" + depLine
		}
		return prefix + depLine + suffix, true
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n[dependencies]\n" + depLine
	return content, true
}

func sanitizeRustCrateName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = invalidRustCrateNameRE.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-_")
	if name == "" {
		return "app"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "app-" + name
	}
	return name
}

func deriveCargoPackageName(wsPath string) string {
	base := filepath.Base(strings.TrimSpace(wsPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "app"
	}
	return sanitizeRustCrateName(base)
}

func minimalCargoTomlBody(packageName string) string {
	packageName = sanitizeRustCrateName(packageName)
	return "[package]\nname = \"" + packageName + "\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n"
}

func workspaceHasRustSources(wsPath string) bool {
	wsPath = strings.TrimSpace(wsPath)
	if wsPath == "" {
		return false
	}
	for _, rel := range []string{"src/main.rs", "src/lib.rs"} {
		if _, err := os.Stat(filepath.Join(wsPath, rel)); err == nil {
			return true
		}
	}
	matches, err := filepath.Glob(filepath.Join(wsPath, "src", "*.rs"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}

func workspaceMissingCargoToml(wsPath string) bool {
	if strings.TrimSpace(wsPath) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(wsPath, "Cargo.toml"))
	return os.IsNotExist(err)
}

func messageImpliesRustGreenfield(content string, hintPaths ...string) bool {
	lower := strings.ToLower(content)
	for _, token := range []string{
		"cargo.toml", "src/main.rs", "src/lib.rs", "cargo build", "cargo run",
		"rust ", "using rust", " in rust", "with rust",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	for _, p := range hintPaths {
		p = normalizeFileChangeRelPath(p)
		if strings.HasSuffix(strings.ToLower(p), ".rs") {
			return true
		}
	}
	return false
}

func refreshRustStackManifest(state *ImplementationSessionState, wsPath string) {
	if state == nil || strings.TrimSpace(wsPath) == "" {
		return
	}
	state.StackManifest = DetectStackManifest(wsPath)
}

func rustMissingCrateEvidence(state *ImplementationSessionState, evidence string) string {
	evidence = strings.TrimSpace(evidence)
	if evidence != "" {
		return evidence
	}
	if state != nil {
		if evidence = strings.TrimSpace(state.VerifyOutput); evidence != "" {
			return evidence
		}
		evidence = strings.TrimSpace(state.LastCommandOutput())
	}
	return evidence
}

// tryGreenfieldCargoTomlScaffold creates a minimal root Cargo.toml when Rust sources or
// intent exist but the manifest is missing (common greenfield implement failure mode).
func (a *Agent) tryGreenfieldCargoTomlScaffold(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState, triggerPath string) bool {
	ok, _ := a.attemptGreenfieldCargoTomlScaffold(ctx, msg, wsPath, msg.Channel, state, triggerPath)
	return ok
}

func (a *Agent) attemptGreenfieldCargoTomlScaffold(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
	triggerPath string,
) (bool, []string) {
	if a == nil || msg == nil || wsPath == "" {
		return false, nil
	}
	if skipCollabCodingFixtureSynths(msg) {
		return false, nil
	}
	if channel == "" {
		channel = "general"
	}
	if !workspaceMissingCargoToml(wsPath) {
		return false, nil
	}
	hasSources := workspaceHasRustSources(wsPath)
	hintPaths := []string{triggerPath}
	if state != nil {
		hintPaths = append(hintPaths, state.FilesChanged...)
		hintPaths = append(hintPaths, state.RegisteredFiles...)
	}
	hasIntent := messageImpliesRustGreenfield(msg.Content, hintPaths...)
	if !hasSources && !hasIntent {
		return false, nil
	}
	body := minimalCargoTomlBody(deriveCargoPackageName(wsPath))
	manifest := a.manifestForProposal(ctx, msg)
	if err := ValidateProposal(wsPath, "Cargo.toml", ProposalOpCreate, manifest); err != nil {
		return false, nil
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["deterministic_edit"] = true
	if err := a.proposeFileCreateInChannel(ctx, channel, "Cargo.toml", body, msg); err != nil {
		return false, nil
	}
	cargoPath := filepath.Join(wsPath, "Cargo.toml")
	onDisk, readErr := os.ReadFile(cargoPath)
	if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
		if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
			return false, nil
		}
		if err := os.WriteFile(cargoPath, []byte(body), 0o644); err != nil {
			return false, nil
		}
		onDisk, readErr = os.ReadFile(cargoPath)
		if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
			return false, nil
		}
		if state != nil {
			state.releaseSnapshot("Cargo.toml")
		}
		log.Printf("[%s] greenfield_cargo_toml_direct_apply", a.Info.Name)
	}
	if state != nil {
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{"Cargo.toml"})
		state.RecordEdit("Cargo.toml")
		state.RecordReadPath("Cargo.toml")
		state.SetPlaybookUsed("greenfield_cargo_toml")
		refreshRustStackManifest(state, wsPath)
	}
	log.Printf("[%s] greenfield_cargo_toml_scaffold(trigger=%s)", a.Info.Name, triggerPath)
	return true, []string{"Cargo.toml"}
}

// tryMissingRustCrateFix adds undeclared external crates to Cargo.toml when cargo build
// reports E0432/E0433 unresolved imports.
func (a *Agent) tryMissingRustCrateFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState, evidence string) bool {
	ok, _ := a.attemptMissingRustCrateFix(ctx, msg, wsPath, msg.Channel, state, evidence)
	return ok
}

func (a *Agent) attemptMissingRustCrateFix(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
	evidence string,
) (bool, []string) {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false, nil
	}
	if channel == "" {
		channel = "general"
	}
	evidence = rustMissingCrateEvidence(state, evidence)
	if evidence == "" {
		evidence = msg.Content
	}
	if commandOutputMatchesPlaybook(evidence) != "rust_missing_crate" {
		return false, nil
	}
	crates := extractMissingRustCrates(evidence)
	if len(crates) == 0 {
		return false, nil
	}
	crate := crates[0]
	cargoPath := filepath.Join(wsPath, "Cargo.toml")
	existing, err := os.ReadFile(cargoPath)
	if err != nil {
		return false, nil
	}
	if cargoTomlHasDependency(string(existing), crate) {
		return false, nil
	}
	body, ok := addMissingDependencyToCargoToml(existing, crate)
	if !ok {
		return false, nil
	}
	oldContent := string(existing)
	if err := a.validateProposalForSession(ctx, msg, "Cargo.toml", ProposalOpEdit); err != nil {
		return false, nil
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["deterministic_edit"] = true
	if err := a.proposeFileEditInChannel(ctx, channel, "Cargo.toml", oldContent, body, msg); err != nil {
		return false, nil
	}
	onDisk, readErr := os.ReadFile(cargoPath)
	if readErr != nil || !cargoTomlHasDependency(string(onDisk), crate) {
		if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
			return false, nil
		}
		if err := os.WriteFile(cargoPath, []byte(body), 0o644); err != nil {
			return false, nil
		}
		onDisk, readErr = os.ReadFile(cargoPath)
		if readErr != nil || !cargoTomlHasDependency(string(onDisk), crate) {
			return false, nil
		}
		state.releaseSnapshot("Cargo.toml")
		log.Printf("[%s] rust_missing_crate_direct_apply(crate=%s)", a.Info.Name, crate)
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{"Cargo.toml"})
	state.RecordEdit("Cargo.toml")
	state.SetPlaybookUsed("rust_missing_crate")
	log.Printf("[%s] rust_missing_crate_fix(crate=%s)", a.Info.Name, crate)
	return true, []string{"Cargo.toml"}
}
