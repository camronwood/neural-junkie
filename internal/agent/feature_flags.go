package agent

import (
	"os"
	"strings"
)

// legacyFileChangeParseEnabled gates fenced/loose [FILE_CHANGE] text parsing (off by default).
func legacyFileChangeParseEnabled() bool {
	v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_LEGACY_FILE_CHANGE_PARSE"))
	return v == "1" || strings.EqualFold(v, "true")
}
