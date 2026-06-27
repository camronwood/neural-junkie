package packs

import "strings"

// OfficialPackIDs are known official domain pack ids (catalog ordering and ID collision checks).
var OfficialPackIDs = []string{"software-development", "life-sciences", "specialist-tuning", "cad", "aws", "incident-management", "web-browser", "music-creation"}

// KnownSpecialistAgentTypes lists in-process specialist agent types gated by domain packs.
var KnownSpecialistAgentTypes = []string{
	"backend", "frontend", "devops", "security", "architecture", "code-review", "database",
	"biology", "cad", "aws", "incident", "browser", "music",
}

// SoftwareDevelopmentExpertSlugs are /create-expert slugs from the software-development pack.
var SoftwareDevelopmentExpertSlugs = []string{
	"backend", "frontend", "devops", "security", "architecture", "code-review", "database",
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
	case "backend", "frontend", "devops", "security", "architecture", "code-review", "database":
		return "software-development"
	case "biology":
		return "life-sciences"
	case "cad":
		return "cad"
	case "aws":
		return "aws"
	case "incident":
		return "incident-management"
	case "browser":
		return "web-browser"
	case "music":
		return "music-creation"
	default:
		return ""
	}
}
