package packs

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// OfficialPackIDs are known official domain pack ids (catalog ordering and ID collision checks).
var OfficialPackIDs = []string{"ide", "software-development", "life-sciences", "specialist-tuning", "cad", "aws", "incident-management", "web-browser", "music-creation", "model-arena", "room-chat", "maps"}

// KnownSpecialistAgentTypes lists in-process specialist agent types gated by domain packs.
var KnownSpecialistAgentTypes = []string{
	"backend", "frontend", "devops", "security", "architecture", "database",
	"rust", "sre", "mobile", "data-ml",
	"biology", "genomics", "structural-biology", "cheminformatics", "cad", "manufacturing", "aws", "incident", "arena",
}

// SoftwareDevelopmentExpertSlugs are /create-expert slugs from the software-development pack.
var SoftwareDevelopmentExpertSlugs = []string{
	"backend", "frontend", "devops", "security", "architecture", "database",
	"rust", "sre", "mobile", "data-ml",
}

// RetiredAbilityPackAgentTypes are former pack specialists now delivered as Assistant abilities
// (or core review behavior for code-review). Kept for config migration only.
var RetiredAbilityPackAgentTypes = []string{
	"maps", "music", "browser", "code-review",
}

// IsRetiredAbilityPackAgentType reports whether agentType is a removed ability-pack specialist.
func IsRetiredAbilityPackAgentType(agentType string) bool {
	t := strings.ToLower(strings.TrimSpace(agentType))
	for _, r := range RetiredAbilityPackAgentTypes {
		if t == r {
			return true
		}
	}
	return false
}

var packIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

// NormalizePackID lowercases and trims a pack id for comparisons and install paths.
func NormalizePackID(packID string) string {
	return strings.ToLower(strings.TrimSpace(packID))
}

// ValidatePackIDFormat rejects empty, path-like, or otherwise unsafe pack ids.
func ValidatePackIDFormat(packID string) error {
	raw := strings.TrimSpace(packID)
	if raw == "" {
		return fmt.Errorf("pack id required")
	}
	if strings.ContainsAny(raw, `/\`) || strings.Contains(raw, "..") || strings.HasPrefix(raw, ".") {
		return fmt.Errorf("pack id %q must not be a relative path", packID)
	}
	id := NormalizePackID(raw)
	if !packIDPattern.MatchString(id) {
		return fmt.Errorf("pack id %q must match %s", packID, packIDPattern.String())
	}
	return nil
}

// IsOfficialPackID reports whether id is a reserved official pack id (case-insensitive).
func IsOfficialPackID(packID string) bool {
	packID = NormalizePackID(packID)
	for _, id := range OfficialPackIDs {
		if id == packID {
			return true
		}
	}
	return false
}

// PackInstallDest returns packsRoot/<normalized-id> after validating id and containment.
func PackInstallDest(packsRoot, packID string) (string, error) {
	if err := ValidatePackIDFormat(packID); err != nil {
		return "", err
	}
	id := NormalizePackID(packID)
	root, err := filepath.Abs(packsRoot)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(root, id)
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, absDest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("pack install path escapes packs directory")
	}
	return absDest, nil
}

// PackIDForAgentType returns the official pack id that owns agentType, or "".
func PackIDForAgentType(agentType string) string {
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "backend", "frontend", "devops", "security", "architecture", "database", "rust", "sre", "mobile", "data-ml":
		return "software-development"
	case "biology", "genomics", "structural-biology", "cheminformatics":
		return "life-sciences"
	case "cad", "manufacturing":
		return "cad"
	case "aws":
		return "aws"
	case "incident":
		return "incident-management"
	case "arena":
		return "model-arena"
	default:
		return ""
	}
}
