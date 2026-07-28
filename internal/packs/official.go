package packs

import "strings"

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

// IsOfficialPackID reports whether id is a reserved official pack id.
func IsOfficialPackID(packID string) bool {
	packID = strings.TrimSpace(packID)
	for _, id := range OfficialPackIDs {
		if id == packID {
			return true
		}
	}
	return false
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
