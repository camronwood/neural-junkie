package collaboration

import (
	"regexp"
	"strings"
)

var mentionStripRe = regexp.MustCompile(`@([a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*)`)

// DeriveCollaborationTitle builds a short human-friendly title from the full goal text.
func DeriveCollaborationTitle(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return "Collaboration"
	}
	// Drop leading slash-command residue.
	if strings.HasPrefix(desc, "/collaborate") {
		parts := strings.Fields(desc)
		if len(parts) > 1 {
			desc = strings.Join(parts[1:], " ")
		}
	}
	desc = mentionStripRe.ReplaceAllString(desc, "")
	desc = strings.TrimSpace(desc)
	// First sentence or line.
	for _, sep := range []string{"\n", ". ", "? ", "! "} {
		if i := strings.Index(desc, sep); i > 0 && i < 72 {
			desc = strings.TrimSpace(desc[:i])
			break
		}
	}
	desc = strings.Join(strings.Fields(desc), " ")
	if desc == "" {
		return "Collaboration"
	}
	return truncate(desc, 72)
}
