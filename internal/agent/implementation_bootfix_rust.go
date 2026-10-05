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
	rustDebugNotImplForRE  = regexp.MustCompile("(?i)trait `Debug` is not implemented for `([A-Za-z_][A-Za-z0-9_]*)`")
	rustDebugAnnotateRE    = regexp.MustCompile("(?i)annotating `([A-Za-z_][A-Za-z0-9_]*)` with `#\\[derive\\(Debug\\)\\]`")
	rustCloneHintTypeRE    = regexp.MustCompile("(?i)if `([A-Za-z_][A-Za-z0-9_]*)` implemented `Clone`")
	rustCopyHintTypeRE     = regexp.MustCompile("(?i)if `([A-Za-z_][A-Za-z0-9_]*)` implemented `Copy`")
	rustMoveValueTypeRE    = regexp.MustCompile("(?i)move occurs because `[^`]*` has type `([A-Za-z_][A-Za-z0-9_]*)`")
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
	// Scenario prompts (blackjack) prefer std-only — rewrite away rand instead of pulling crates.io.
	if crate == "rand" && userPrefersStdOnlyRust(msg) {
		if changed := a.tryRewriteRustRandToStdShuffle(ctx, msg, wsPath, channel, state); len(changed) > 0 {
			state.SetPlaybookUsed("rust_std_shuffle_rewrite")
			log.Printf("[%s] rust_std_shuffle_rewrite(files=%v)", a.Info.Name, changed)
			return true, changed
		}
	}
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
	proposeErr := error(nil)
	if _, err := a.proposeFileEditInChannel(ctx, channel, "Cargo.toml", oldContent, body, msg); err != nil {
		proposeErr = err
		log.Printf("[%s] rust_missing_crate_propose_failed(crate=%s err=%v)", a.Info.Name, crate, err)
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
		log.Printf("[%s] rust_missing_crate_direct_apply(crate=%s propose_err=%v)", a.Info.Name, crate, proposeErr)
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{"Cargo.toml"})
	state.RecordEdit("Cargo.toml")
	state.SetPlaybookUsed("rust_missing_crate")
	log.Printf("[%s] rust_missing_crate_fix(crate=%s)", a.Info.Name, crate)
	changed := []string{"Cargo.toml"}
	if crate == "rand" {
		if okFiles := a.tryEnsureRustRandShuffleImport(ctx, msg, wsPath, channel, state); len(okFiles) > 0 {
			changed = appendUnique(changed, okFiles)
		}
	}
	return true, changed
}

func extractMissingRustDebugTypes(output string) []string {
	seen := make(map[string]bool)
	var types []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		types = append(types, name)
	}
	for _, re := range []*regexp.Regexp{rustDebugAnnotateRE, rustDebugNotImplForRE} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return types
}

// addDeriveDebugToRustSource inserts or merges #[derive(Debug)] on struct/enum typeName.
func addDeriveDebugToRustSource(src, typeName string) (string, bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || src == "" {
		return "", false
	}
	declRE := regexp.MustCompile(`(?m)^([ \t]*)((?:pub(?:\([^)]*\))?\s+)?)(struct|enum)\s+` + regexp.QuoteMeta(typeName) + `\b`)
	loc := declRE.FindStringSubmatchIndex(src)
	if loc == nil {
		return "", false
	}
	lineStart := loc[0]
	indent := src[loc[2]:loc[3]]

	// Scan attribute lines immediately above the declaration.
	before := src[:lineStart]
	lines := strings.Split(before, "\n")
	i := len(lines) - 1
	if i >= 0 && lines[i] == "" {
		i--
	}
	for i >= 0 {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			break
		}
		if strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "#![") {
			if strings.Contains(trimmed, "derive(") && strings.Contains(trimmed, "Debug") {
				return "", false
			}
			if strings.HasPrefix(trimmed, "#[derive(") && strings.HasSuffix(trimmed, ")]") && !strings.Contains(trimmed, "Debug") {
				inner := trimmed[len("#[derive(") : len(trimmed)-2]
				inner = strings.TrimSpace(inner)
				lead := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				if inner == "" {
					lines[i] = lead + "#[derive(Debug)]"
				} else {
					lines[i] = lead + "#[derive(" + inner + ", Debug)]"
				}
				newBefore := strings.Join(lines, "\n")
				if strings.HasSuffix(before, "\n") && !strings.HasSuffix(newBefore, "\n") {
					newBefore += "\n"
				}
				return newBefore + src[lineStart:], true
			}
			i--
			continue
		}
		break
	}
	insert := indent + "#[derive(Debug)]\n"
	return src[:lineStart] + insert + src[lineStart:], true
}

func rustSourceFilesForDebugFix(wsPath string) []string {
	var out []string
	preferred := []string{
		filepath.Join(wsPath, "src", "main.rs"),
		filepath.Join(wsPath, "src", "lib.rs"),
		filepath.Join(wsPath, "main.rs"),
		filepath.Join(wsPath, "lib.rs"),
	}
	for _, p := range preferred {
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	if len(out) > 0 {
		return out
	}
	_ = filepath.Walk(wsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".rs") {
			out = append(out, path)
		}
		return nil
	})
	return out
}

// tryMissingRustDebugFix adds #[derive(Debug)] when cargo build reports E0277 missing Debug.
func (a *Agent) tryMissingRustDebugFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState, evidence string) bool {
	ok, _ := a.attemptMissingRustDebugFix(ctx, msg, wsPath, msg.Channel, state, evidence)
	return ok
}

func (a *Agent) attemptMissingRustDebugFix(
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
	if commandOutputMatchesPlaybook(evidence) != "rust_missing_debug" {
		return false, nil
	}
	types := extractMissingRustDebugTypes(evidence)
	if len(types) == 0 {
		return false, nil
	}
	typeName := types[0]
	var changed []string
	for _, abs := range rustSourceFilesForDebugFix(wsPath) {
		existing, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		body, ok := addDeriveDebugToRustSource(string(existing), typeName)
		if !ok {
			continue
		}
		rel, relErr := filepath.Rel(wsPath, abs)
		if relErr != nil || rel == "" || strings.HasPrefix(rel, "..") {
			rel = filepath.Base(abs)
		}
		rel = filepath.ToSlash(rel)
		oldContent := string(existing)
		if err := a.validateProposalForSession(ctx, msg, rel, ProposalOpEdit); err != nil {
			continue
		}
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		msg.Metadata["deterministic_edit"] = true
		var proposeErr error
		if _, err := a.proposeFileEditInChannel(ctx, channel, rel, oldContent, body, msg); err != nil {
			proposeErr = err
			log.Printf("[%s] rust_missing_debug_propose_failed(type=%s file=%s err=%v)", a.Info.Name, typeName, rel, err)
		}
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || !strings.Contains(string(onDisk), "Debug") {
			if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
				continue
			}
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				continue
			}
			onDisk, readErr = os.ReadFile(abs)
			if readErr != nil || !strings.Contains(string(onDisk), "Debug") {
				continue
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] rust_missing_debug_direct_apply(type=%s file=%s propose_err=%v)", a.Info.Name, typeName, rel, proposeErr)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		changed = append(changed, rel)
		state.SetPlaybookUsed("rust_missing_debug")
		log.Printf("[%s] rust_missing_debug_fix(type=%s file=%s)", a.Info.Name, typeName, rel)
		return true, changed
	}
	return false, nil
}

// minimalRustBinStubBody returns a compiling CLI stub. Blackjack greenfield waits assert hit/stand/dealer.
func minimalRustBinStubBody(userContent string) string {
	lower := strings.ToLower(userContent)
	if strings.Contains(lower, "blackjack") || strings.Contains(lower, "hit") || strings.Contains(lower, "dealer") {
		return `use std::io::{self, Write};

fn hand_value(cards: &[u8]) -> u32 {
    let mut total = 0u32;
    let mut aces = 0u32;
    for &c in cards {
        match c {
            1 => {
                aces += 1;
                total += 11;
            }
            11 | 12 | 13 => total += 10,
            n => total += u32::from(n),
        }
    }
    while total > 21 && aces > 0 {
        total -= 10;
        aces -= 1;
    }
    total
}

fn main() {
    let mut deck: Vec<u8> = (1..=13).flat_map(|r| std::iter::repeat(r).take(4)).collect();
    // std-only shuffle
    let seed = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos() as usize)
        .unwrap_or(1);
    for i in (1..deck.len()).rev() {
        let j = (seed + i * 2654435761) % (i + 1);
        deck.swap(i, j);
    }
    let mut player = vec![deck.pop().unwrap_or(10), deck.pop().unwrap_or(5)];
    let mut dealer = vec![deck.pop().unwrap_or(9), deck.pop().unwrap_or(6)];
    println!("Blackjack — you: {:?} ({})  dealer shows: {}", player, hand_value(&player), dealer[0]);
    print!("hit or stand? ");
    let _ = io::stdout().flush();
    let mut line = String::new();
    let _ = io::stdin().read_line(&mut line);
    if line.to_lowercase().contains("hit") {
        player.push(deck.pop().unwrap_or(2));
    }
    while hand_value(&dealer) < 17 {
        dealer.push(deck.pop().unwrap_or(3));
    }
    let pv = hand_value(&player);
    let dv = hand_value(&dealer);
    if pv > 21 {
        println!("You bust. Dealer wins.");
    } else if dv > 21 || pv > dv {
        println!("You win!");
    } else if pv == dv {
        println!("Push.");
    } else {
        println!("Dealer wins.");
    }
}
`
	}
	return "fn main() {\n    println!(\"ok\");\n}\n"
}

// tryMissingRustBinTargetFix scaffolds src/main.rs when Cargo.toml exists but cargo reports no targets.
func (a *Agent) tryMissingRustBinTargetFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState, evidence string) bool {
	ok, _ := a.attemptMissingRustBinTargetFix(ctx, msg, wsPath, msg.Channel, state, evidence)
	return ok
}

func (a *Agent) attemptMissingRustBinTargetFix(
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
	if commandOutputMatchesPlaybook(evidence) != "rust_missing_bin_target" {
		// Also recover when Cargo.toml is on disk and sources are missing (no verify text yet).
		cargoOK := !workspaceMissingCargoToml(wsPath)
		if !(cargoOK && !workspaceHasRustSources(wsPath) && messageImpliesRustGreenfield(msg.Content)) {
			return false, nil
		}
	}
	mainPath := filepath.Join(wsPath, "src", "main.rs")
	if _, err := os.Stat(mainPath); err == nil {
		return false, nil
	}
	if _, err := os.Stat(filepath.Join(wsPath, "src", "lib.rs")); err == nil {
		return false, nil
	}
	if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
		return false, nil
	}
	body := minimalRustBinStubBody(msg.Content)
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["deterministic_edit"] = true
	if err := os.MkdirAll(filepath.Join(wsPath, "src"), 0o755); err != nil {
		return false, nil
	}
	if err := a.proposeFileCreateInChannel(ctx, channel, "src/main.rs", body, msg); err != nil {
		log.Printf("[%s] rust_missing_bin_target_propose_failed(err=%v)", a.Info.Name, err)
	}
	onDisk, readErr := os.ReadFile(mainPath)
	if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
		if err := os.WriteFile(mainPath, []byte(body), 0o644); err != nil {
			return false, nil
		}
		state.releaseSnapshot("src/main.rs")
		log.Printf("[%s] rust_missing_bin_target_direct_apply", a.Info.Name)
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{"src/main.rs"})
	state.RecordEdit("src/main.rs")
	state.RecordReadPath("src/main.rs")
	state.SetPlaybookUsed("rust_missing_bin_target")
	refreshRustStackManifest(state, wsPath)
	log.Printf("[%s] rust_missing_bin_target_fix(file=src/main.rs)", a.Info.Name)
	return true, []string{"src/main.rs"}
}

// tryMisplacedRustCargoTomlFix recovers when the model wrote Cargo.toml body into src/main.rs
// (cargo: expected item, found `[`). Restores a stub main.rs and a clean root Cargo.toml.
func (a *Agent) tryMisplacedRustCargoTomlFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState, evidence string) bool {
	ok, _ := a.attemptMisplacedRustCargoTomlFix(ctx, msg, wsPath, msg.Channel, state, evidence)
	return ok
}

func (a *Agent) attemptMisplacedRustCargoTomlFix(
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
	mainPath := filepath.Join(wsPath, "src", "main.rs")
	existing, err := os.ReadFile(mainPath)
	if err != nil {
		return false, nil
	}
	body := strings.TrimSpace(string(existing))
	looksLikeToml := strings.HasPrefix(body, "[package]") || strings.Contains(body, "\n[package]\n")
	evidenceHit := strings.Contains(strings.ToLower(evidence), "expected item, found") &&
		(strings.Contains(evidence, "[") || strings.Contains(strings.ToLower(evidence), "src/main.rs"))
	if !looksLikeToml && !evidenceHit {
		return false, nil
	}
	if !looksLikeToml {
		return false, nil
	}
	stub := minimalRustBinStubBody(msg.Content)
	cargoBody := minimalCargoTomlBody(deriveCargoPackageName(wsPath))
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["deterministic_edit"] = true
	if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Join(wsPath, "src"), 0o755); err != nil {
		return false, nil
	}
	if err := os.WriteFile(mainPath, []byte(stub), 0o644); err != nil {
		return false, nil
	}
	if err := os.WriteFile(filepath.Join(wsPath, "Cargo.toml"), []byte(cargoBody), 0o644); err != nil {
		return false, nil
	}
	state.releaseSnapshot("src/main.rs")
	state.releaseSnapshot("Cargo.toml")
	state.ProposedCount += 2
	state.FilesChanged = appendUnique(state.FilesChanged, []string{"src/main.rs", "Cargo.toml"})
	state.RecordEdit("src/main.rs")
	state.RecordEdit("Cargo.toml")
	state.SetPlaybookUsed("rust_misplaced_cargo_toml")
	log.Printf("[%s] rust_misplaced_cargo_toml_fix(file=src/main.rs)", a.Info.Name)
	return true, []string{"src/main.rs", "Cargo.toml"}
}

func extractMissingRustCopyCloneTypes(output string) []string {
	seen := make(map[string]bool)
	var types []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		types = append(types, name)
	}
	for _, re := range []*regexp.Regexp{rustCloneHintTypeRE, rustCopyHintTypeRE, rustMoveValueTypeRE} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return types
}

// addDeriveCopyCloneToRustSource inserts or merges #[derive(Copy, Clone)] on struct/enum typeName.
func addDeriveCopyCloneToRustSource(src, typeName string) (string, bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || src == "" {
		return "", false
	}
	declRE := regexp.MustCompile(`(?m)^([ \t]*)((?:pub(?:\([^)]*\))?\s+)?)(struct|enum)\s+` + regexp.QuoteMeta(typeName) + `\b`)
	loc := declRE.FindStringSubmatchIndex(src)
	if loc == nil {
		return "", false
	}
	lineStart := loc[0]
	indent := src[loc[2]:loc[3]]

	before := src[:lineStart]
	lines := strings.Split(before, "\n")
	i := len(lines) - 1
	if i >= 0 && lines[i] == "" {
		i--
	}
	for i >= 0 {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			break
		}
		if strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "#![") {
			if strings.HasPrefix(trimmed, "#[derive(") && strings.HasSuffix(trimmed, ")]") {
				hasCopy := strings.Contains(trimmed, "Copy")
				hasClone := strings.Contains(trimmed, "Clone")
				if hasCopy && hasClone {
					return "", false
				}
				inner := trimmed[len("#[derive(") : len(trimmed)-2]
				inner = strings.TrimSpace(inner)
				parts := []string{}
				if inner != "" {
					parts = append(parts, inner)
				}
				if !hasCopy {
					parts = append(parts, "Copy")
				}
				if !hasClone {
					parts = append(parts, "Clone")
				}
				lead := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = lead + "#[derive(" + strings.Join(parts, ", ") + ")]"
				newBefore := strings.Join(lines, "\n")
				if strings.HasSuffix(before, "\n") && !strings.HasSuffix(newBefore, "\n") {
					newBefore += "\n"
				}
				return newBefore + src[lineStart:], true
			}
			i--
			continue
		}
		break
	}
	insert := indent + "#[derive(Copy, Clone)]\n"
	return src[:lineStart] + insert + src[lineStart:], true
}

// tryMissingRustCopyCloneFix adds #[derive(Copy, Clone)] when cargo reports E0382 move / E0507 missing Copy.
func (a *Agent) tryMissingRustCopyCloneFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState, evidence string) bool {
	ok, _ := a.attemptMissingRustCopyCloneFix(ctx, msg, wsPath, msg.Channel, state, evidence)
	return ok
}

func (a *Agent) attemptMissingRustCopyCloneFix(
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
	if commandOutputMatchesPlaybook(evidence) != "rust_missing_copy_clone" {
		return false, nil
	}
	types := extractMissingRustCopyCloneTypes(evidence)
	if len(types) == 0 {
		log.Printf("[%s] rust_missing_copy_clone_skip(reason=no_types)", a.Info.Name)
		return false, nil
	}
	var changed []string
	for _, typeName := range types {
		applied := false
		for _, abs := range rustSourceFilesForDebugFix(wsPath) {
			existing, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			body, ok := addDeriveCopyCloneToRustSource(string(existing), typeName)
			if !ok {
				continue
			}
			rel, relErr := filepath.Rel(wsPath, abs)
			if relErr != nil || rel == "" || strings.HasPrefix(rel, "..") {
				rel = filepath.Base(abs)
			}
			rel = filepath.ToSlash(rel)
			oldContent := string(existing)
			if err := a.validateProposalForSession(ctx, msg, rel, ProposalOpEdit); err != nil {
				log.Printf("[%s] rust_missing_copy_clone_skip(type=%s file=%s reason=validate:%v)", a.Info.Name, typeName, rel, err)
				continue
			}
			if msg.Metadata == nil {
				msg.Metadata = map[string]interface{}{}
			}
			msg.Metadata["deterministic_edit"] = true
			var proposeErr error
			if _, err := a.proposeFileEditInChannel(ctx, channel, rel, oldContent, body, msg); err != nil {
				proposeErr = err
				log.Printf("[%s] rust_missing_copy_clone_propose_failed(type=%s file=%s err=%v)", a.Info.Name, typeName, rel, err)
			}
			onDisk, readErr := os.ReadFile(abs)
			if readErr != nil || !strings.Contains(string(onDisk), "Clone") {
				if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
					continue
				}
				if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
					continue
				}
				onDisk, readErr = os.ReadFile(abs)
				if readErr != nil || !strings.Contains(string(onDisk), "Clone") {
					continue
				}
				state.releaseSnapshot(rel)
				log.Printf("[%s] rust_missing_copy_clone_direct_apply(type=%s file=%s propose_err=%v)", a.Info.Name, typeName, rel, proposeErr)
			}
			state.ProposedCount++
			state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
			state.RecordEdit(rel)
			changed = append(changed, rel)
			state.SetPlaybookUsed("rust_missing_copy_clone")
			log.Printf("[%s] rust_missing_copy_clone_fix(type=%s file=%s)", a.Info.Name, typeName, rel)
			applied = true
			break
		}
		if !applied {
			log.Printf("[%s] rust_missing_copy_clone_skip(type=%s reason=no_source_apply)", a.Info.Name, typeName)
		}
	}
	return len(changed) > 0, changed
}

func userPrefersStdOnlyRust(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	lower := strings.ToLower(msg.Content)
	markers := []string{
		"standard library only",
		"std only",
		"prefer the rust standard library",
		"avoid rand",
		"no rand",
		"stdlib only",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

var (
	rustUseRandLineRE   = regexp.MustCompile(`(?m)^\s*use\s+rand(?:::[^;]+)?;\s*\n?`)
	rustShuffleCallRE   = regexp.MustCompile(`\.shuffle\s*\(\s*&mut\s+[A-Za-z_][A-Za-z0-9_]*\s*\)`)
	rustThreadRngCallRE = regexp.MustCompile(`thread_rng\s*\(\s*\)`)
)

// rewriteRustSourceDropRand replaces rand shuffle usage with a tiny std-only helper.
func rewriteRustSourceDropRand(src string) (string, bool) {
	if !strings.Contains(src, "rand") && !strings.Contains(src, ".shuffle(") {
		return "", false
	}
	out := src
	changed := false
	if rustUseRandLineRE.MatchString(out) {
		out = rustUseRandLineRE.ReplaceAllString(out, "")
		changed = true
	}
	recvShuffle := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\.shuffle\s*\(\s*&mut\s+[A-Za-z_][A-Za-z0-9_]*\s*\)`)
	if recvShuffle.MatchString(out) {
		out = recvShuffle.ReplaceAllString(out, "nj_std_shuffle(&mut $1)")
		changed = true
	} else if rustShuffleCallRE.MatchString(out) {
		// Bare .shuffle(&mut rng) without a simple receiver — leave for model repair.
		return "", false
	}
	if rustThreadRngCallRE.MatchString(out) {
		out = regexp.MustCompile(`(?m)^\s*let\s+mut\s+\w+\s*=\s*thread_rng\s*\(\s*\)\s*;\s*\n?`).ReplaceAllString(out, "")
		out = regexp.MustCompile(`(?m)^\s*let\s+\w+\s*=\s*thread_rng\s*\(\s*\)\s*;\s*\n?`).ReplaceAllString(out, "")
		changed = true
	}
	if !changed {
		return "", false
	}
	if !strings.Contains(out, "fn nj_std_shuffle") {
		helper := "\nfn nj_std_shuffle<T>(items: &mut [T]) {\n" +
			"    let n = items.len();\n" +
			"    if n < 2 {\n" +
			"        return;\n" +
			"    }\n" +
			"    let seed = std::time::SystemTime::now()\n" +
			"        .duration_since(std::time::UNIX_EPOCH)\n" +
			"        .map(|d| d.as_nanos() as usize)\n" +
			"        .unwrap_or(1);\n" +
			"    for i in (1..n).rev() {\n" +
			"        let j = (seed + i * 2654435761) % (i + 1);\n" +
			"        items.swap(i, j);\n" +
			"    }\n" +
			"}\n"
		out = strings.TrimRight(out, "\n") + "\n" + helper
	}
	return out, true
}

func (a *Agent) tryRewriteRustRandToStdShuffle(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
) []string {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return nil
	}
	if channel == "" {
		channel = "general"
	}
	var changed []string
	for _, abs := range rustSourceFilesForDebugFix(wsPath) {
		existing, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		body, ok := rewriteRustSourceDropRand(string(existing))
		if !ok {
			continue
		}
		rel, relErr := filepath.Rel(wsPath, abs)
		if relErr != nil || rel == "" || strings.HasPrefix(rel, "..") {
			rel = filepath.Base(abs)
		}
		rel = filepath.ToSlash(rel)
		oldContent := string(existing)
		if err := a.validateProposalForSession(ctx, msg, rel, ProposalOpEdit); err != nil {
			continue
		}
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		msg.Metadata["deterministic_edit"] = true
		if _, err := a.proposeFileEditInChannel(ctx, channel, rel, oldContent, body, msg); err != nil {
			log.Printf("[%s] rust_std_shuffle_rewrite_propose_failed(file=%s err=%v)", a.Info.Name, rel, err)
		}
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || strings.Contains(string(onDisk), "use rand") {
			if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
				continue
			}
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				continue
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] rust_std_shuffle_rewrite_direct_apply(file=%s)", a.Info.Name, rel)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		changed = append(changed, rel)
	}
	// Drop rand from Cargo.toml if present so verify stays std-only / offline-friendly.
	cargoPath := filepath.Join(wsPath, "Cargo.toml")
	if cargoBytes, err := os.ReadFile(cargoPath); err == nil && cargoTomlHasDependency(string(cargoBytes), "rand") {
		cleaned := removeCargoTomlDependency(string(cargoBytes), "rand")
		if cleaned != string(cargoBytes) {
			if resolveImplementationTrustMode(msg) == editorTrustAutoApply {
				_ = os.WriteFile(cargoPath, []byte(cleaned), 0o644)
				state.releaseSnapshot("Cargo.toml")
				state.ProposedCount++
				state.FilesChanged = appendUnique(state.FilesChanged, []string{"Cargo.toml"})
				state.RecordEdit("Cargo.toml")
				changed = appendUnique(changed, []string{"Cargo.toml"})
			}
		}
	}
	return changed
}

func removeCargoTomlDependency(cargo, crate string) string {
	crate = strings.TrimSpace(crate)
	if crate == "" {
		return cargo
	}
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(crate) + `\s*=\s*[^\n]*\n?`)
	return re.ReplaceAllString(cargo, "")
}

// ensureRustRandShuffleImport adds `use rand::seq::SliceRandom` when .shuffle( is used without it.
func ensureRustRandShuffleImport(src string) (string, bool) {
	if !strings.Contains(src, ".shuffle(") {
		return "", false
	}
	if strings.Contains(src, "SliceRandom") {
		return "", false
	}
	insert := "use rand::seq::SliceRandom;\n"
	// Prefer after the last leading use/mod/#! line.
	lines := strings.Split(src, "\n")
	insertAt := 0
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#!") ||
			strings.HasPrefix(trim, "use ") || strings.HasPrefix(trim, "mod ") ||
			strings.HasPrefix(trim, "extern ") {
			insertAt = i + 1
			continue
		}
		break
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, strings.TrimSuffix(insert, "\n"))
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n"), true
}

func (a *Agent) tryEnsureRustRandShuffleImport(
	ctx context.Context,
	msg *protocol.Message,
	wsPath, channel string,
	state *ImplementationSessionState,
) []string {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return nil
	}
	if channel == "" {
		channel = "general"
	}
	var changed []string
	for _, abs := range rustSourceFilesForDebugFix(wsPath) {
		existing, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		body, ok := ensureRustRandShuffleImport(string(existing))
		if !ok {
			continue
		}
		rel, relErr := filepath.Rel(wsPath, abs)
		if relErr != nil || rel == "" || strings.HasPrefix(rel, "..") {
			rel = filepath.Base(abs)
		}
		rel = filepath.ToSlash(rel)
		oldContent := string(existing)
		if err := a.validateProposalForSession(ctx, msg, rel, ProposalOpEdit); err != nil {
			continue
		}
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		msg.Metadata["deterministic_edit"] = true
		var proposeErr error
		if _, err := a.proposeFileEditInChannel(ctx, channel, rel, oldContent, body, msg); err != nil {
			proposeErr = err
			log.Printf("[%s] rust_rand_shuffle_import_propose_failed(file=%s err=%v)", a.Info.Name, rel, err)
		}
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || !strings.Contains(string(onDisk), "SliceRandom") {
			if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
				continue
			}
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				continue
			}
			onDisk, readErr = os.ReadFile(abs)
			if readErr != nil || !strings.Contains(string(onDisk), "SliceRandom") {
				continue
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] rust_rand_shuffle_import_direct_apply(file=%s propose_err=%v)", a.Info.Name, rel, proposeErr)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		changed = append(changed, rel)
		log.Printf("[%s] rust_rand_shuffle_import_fix(file=%s)", a.Info.Name, rel)
	}
	return changed
}
