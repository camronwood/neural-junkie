package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var musicGenerationVerbRE = regexp.MustCompile(`\b(make|generate|create|compose|produce|write)\b`)

// normalizeMusicRequestText fixes common typos in music-generation asks.
func normalizeMusicRequestText(content string) string {
	c := strings.ToLower(content)
	repl := strings.NewReplacer(
		"genereate", "generate",
		"generete", "generate",
		"genrate", "generate",
		"cretae", "create",
		"compse", "compose",
	)
	return repl.Replace(c)
}

// UserRequestsGeneratedMusic is a lightweight heuristic for music-generation asks.
func UserRequestsGeneratedMusic(content string) bool {
	c := normalizeMusicRequestText(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	if c == strings.ToLower(protocol.GeneratedAudioDeliveryContent) {
		return false
	}
	phrases := []string{
		"generate a song", "generate me a song", "generate me some music",
		"create a song", "create me a song", "make a song", "make me a song",
		"compose a song", "compose me a song", "produce a song", "write a song",
		"generate music", "create music", "make music", "compose music",
		"generate an instrumental", "create an instrumental", "make an instrumental",
		"generate a track", "create a track",
	}
	for _, p := range phrases {
		if strings.Contains(c, p) {
			return true
		}
	}
	hasNoun := strings.Contains(c, "song") || strings.Contains(c, "music") ||
		strings.Contains(c, "instrumental") || strings.Contains(c, "track")
	return hasNoun && musicGenerationVerbRE.MatchString(c)
}

// MusicStyleTagsFromMessage strips mentions and request boilerplate; returns style tags or "".
func MusicStyleTagsFromMessage(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	for strings.HasPrefix(content, "@") {
		if i := strings.IndexByte(content, ' '); i > 0 {
			content = strings.TrimSpace(content[i+1:])
		} else {
			break
		}
	}
	remaining := stripMusicRequestBoilerplate(content)
	if len(remaining) < 4 {
		return ""
	}
	if len(remaining) > 900 {
		remaining = remaining[:900]
	}
	return remaining
}

func stripMusicRequestBoilerplate(content string) string {
	c := strings.ToLower(strings.TrimSpace(content))
	for _, prefix := range []string{"can you ", "could you ", "please ", "would you ", "will you "} {
		c = strings.TrimPrefix(c, prefix)
	}
	for _, phrase := range []string{
		"generate me a song about ", "generate me a song ", "generate a song about ", "generate a song ",
		"generate me some music about ", "generate me some music ", "generate music about ", "generate music ",
		"create me a song about ", "create me a song ", "create a song about ", "create a song ",
		"make me a song about ", "make me a song ", "make a song about ", "make a song ",
		"compose me a song about ", "compose me a song ", "compose a song about ", "compose a song ",
		"produce a song about ", "produce a song ", "write a song about ", "write a song ",
		"generate an instrumental about ", "generate an instrumental ",
		"create an instrumental about ", "create an instrumental ",
		"generate a track about ", "generate a track ",
	} {
		if strings.HasPrefix(c, phrase) {
			c = strings.TrimSpace(c[len(phrase):])
			break
		}
	}
	for _, phrase := range []string{
		"generate me a song", "generate a song", "generate me some music", "generate music",
		"create me a song", "create a song", "make me a song", "make a song",
		"compose me a song", "compose a song", "produce a song", "write a song",
		"generate an instrumental", "create an instrumental", "generate a track", "create a track",
	} {
		c = strings.ReplaceAll(c, phrase, " ")
	}
	c = strings.TrimSpace(c)
	c = strings.Trim(c, "?.!")
	if c == "" {
		return ""
	}
	// Preserve original casing from content where possible by using trimmed tail.
	if idx := strings.Index(strings.ToLower(content), c); idx >= 0 {
		return strings.TrimSpace(content[idx : idx+len(c)])
	}
	return c
}

// DefaultMusicStyleTags returns ACE-Step style tags when the user did not specify a mood or genre.
func DefaultMusicStyleTags() string {
	return "upbeat indie pop, 110 bpm, warm male vocal, acoustic guitar, bright drums, modern production"
}

func musicRequestWantsVocals(content string) bool {
	c := strings.ToLower(content)
	if strings.Contains(c, "instrumental") {
		return false
	}
	for _, p := range []string{"with lyrics", "with vocals", "with vocal", "with singing", "write lyrics"} {
		if strings.Contains(c, p) {
			return true
		}
	}
	return false
}
