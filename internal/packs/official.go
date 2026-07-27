package packs

import "strings"

// OfficialPackIDs are known official domain pack ids (catalog ordering and ID collision checks).
var OfficialPackIDs = []string{"ide", "software-development", "life-sciences", "specialist-tuning", "cad", "aws", "incident-management", "web-browser", "music-creation", "model-arena", "room-chat", "maps"}

// KnownSpecialistAgentTypes lists in-process specialist agent types gated by domain packs.
var KnownSpecialistAgentTypes = []string{
	"backend", "frontend", "devops", "security", "architecture", "code-review", "database",
	"rust", "sre", "mobile", "data-ml",
	"biology", "genomics", "structural-biology", "cheminformatics", "cad", "manufacturing", "aws", "incident", "browser", "music", "arena", "maps",
}

// SoftwareDevelopmentExpertSlugs are /create-expert slugs from the software-development pack.
var SoftwareDevelopmentExpertSlugs = []string{
	"backend", "frontend", "devops", "security", "architecture", "code-review", "database",
	"rust", "sre", "mobile", "data-ml",
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
	case "backend", "frontend", "devops", "security", "architecture", "code-review", "database", "rust", "sre", "mobile", "data-ml":
		return "software-development"
	case "biology", "genomics", "structural-biology", "cheminformatics":
		return "life-sciences"
	case "cad", "manufacturing":
		return "cad"
	case "aws":
		return "aws"
	case "incident":
		return "incident-management"
	case "browser":
		return "web-browser"
	case "music":
		return "music-creation"
	case "arena":
		return "model-arena"
	case "maps":
		return "maps"
	default:
		return ""
	}
}
