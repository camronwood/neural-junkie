package agent

import (
	"os/exec"
	"strings"
)

// CLICommandCandidates returns primary and alternate binary names to probe on PATH (deduped).
func CLICommandCandidates(cfg CLIAgentConfig) []string {
	seen := make(map[string]bool)
	var out []string
	for _, c := range append([]string{cfg.Command}, cfg.AlternateCommands...) {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// ResolveCLICommand returns the first CLI binary found on PATH for this config.
func ResolveCLICommand(cfg CLIAgentConfig) (command string, ok bool) {
	for _, c := range CLICommandCandidates(cfg) {
		if _, err := exec.LookPath(c); err == nil {
			return c, true
		}
	}
	return "", false
}

// EffectiveBaseArgs returns invoke args for the resolved binary (legacy Copilot uses no -p).
func EffectiveBaseArgs(cfg CLIAgentConfig, resolvedCommand string) []string {
	if cfg.AlternateBaseArgs != nil {
		if args, ok := cfg.AlternateBaseArgs[resolvedCommand]; ok {
			return append([]string(nil), args...)
		}
	}
	if len(cfg.BaseArgs) == 0 {
		return nil
	}
	return append([]string(nil), cfg.BaseArgs...)
}

// ResolvedCLI holds the binary and args to pass to CLIAgentProvider.
type ResolvedCLI struct {
	Command  string
	BaseArgs []string
}

// ResolveCLI resolves PATH and base args for a registry entry.
func ResolveCLI(cfg CLIAgentConfig) (ResolvedCLI, bool) {
	cmd, ok := ResolveCLICommand(cfg)
	if !ok {
		return ResolvedCLI{}, false
	}
	return ResolvedCLI{
		Command:  cmd,
		BaseArgs: EffectiveBaseArgs(cfg, cmd),
	}, true
}

// CLIProbeLabel formats candidate names for logs and errors.
func CLIProbeLabel(cfg CLIAgentConfig) string {
	cands := CLICommandCandidates(cfg)
	if len(cands) == 0 {
		return cfg.Command
	}
	return strings.Join(cands, " | ")
}
