package agent

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	semantic "github.com/camronwood/neural-junkie/internal/intent"
)

const maxResponseImageBytes = 8 * 1024 * 1024

// absoluteImagePathRE matches Unix paths anywhere (handles markdown **/path** and backticks).
var absoluteImagePathRE = regexp.MustCompile(`(?i)(/[\w./\-~]+\.(?:png|jpe?g|gif|webp))`)

// relativeImagePathRE matches filenames when the hub can resolve them under CLI work dirs.
var relativeImagePathRE = regexp.MustCompile(`(?i)(?:^|[\s"'(])([\w./\-]+\.(?:png|jpe?g|gif|webp))`)

// AttachGeneratedImageFromResponse scans agent text for local image files and sets generated_image metadata.
// Returns true when an image was attached.
func AttachGeneratedImageFromResponse(msg *protocol.Message, searchDirs ...string) bool {
	if msg == nil {
		return false
	}
	if _, ok := msg.Metadata["generated_image"]; ok {
		return true
	}
	for _, p := range extractImagePathsFromText(msg.Content) {
		resolved := resolveExistingImagePath(p, searchDirs)
		if resolved == "" {
			continue
		}
		b, mime, err := readLocalImage(resolved)
		if err != nil {
			log.Printf("[agent] skip image %s: %v", resolved, err)
			continue
		}
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}
		msg.Metadata["generated_image"] = map[string]interface{}{
			"mime": mime,
			"data": base64.StdEncoding.EncodeToString(b),
			"path": resolved,
		}
		log.Printf("[agent] attached generated_image from %s (%d bytes)", resolved, len(b))
		return true
	}
	return false
}

// extractImagePathsFromText finds image file paths in agent prose (markdown-safe).
func extractImagePathsFromText(content string) []string {
	content = stripMarkdownForImageScan(content)
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(strings.Trim(p, `"'`+"`"))
		if p == "" {
			return
		}
		if strings.HasPrefix(p, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(home, strings.TrimPrefix(p, "~/"))
			}
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, m := range absoluteImagePathRE.FindAllString(content, -1) {
		add(m)
	}
	for _, raw := range relativeImagePathRE.FindAllStringSubmatch(content, -1) {
		if len(raw) >= 2 {
			add(raw[1])
		}
	}
	return out
}

func stripMarkdownForImageScan(s string) string {
	s = strings.ReplaceAll(s, "**", " ")
	s = strings.ReplaceAll(s, "__", " ")
	s = strings.ReplaceAll(s, "`", " ")
	return s
}

func resolveExistingImagePath(p string, searchDirs []string) string {
	if filepath.IsAbs(p) {
		if fileExists(p) {
			return filepath.Clean(p)
		}
		return ""
	}
	candidates := []string{p}
	for _, dir := range searchDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(dir, p))
	}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func readLocalImage(path string) ([]byte, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if len(b) == 0 {
		return nil, "", fmt.Errorf("empty file")
	}
	if len(b) > maxResponseImageBytes {
		return nil, "", fmt.Errorf("image exceeds %d bytes", maxResponseImageBytes)
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return b, "image/jpeg", nil
	case ".gif":
		return b, "image/gif", nil
	case ".webp":
		return b, "image/webp", nil
	default:
		return b, "image/png", nil
	}
}

var (
	imageGenerationVerbRE = regexp.MustCompile(`\b(make|generate|create|draw|show|render|design|illustrate|mock\s*up)\b`)
	imageGenerationNounRE = regexp.MustCompile(
		`\b(image|picture|illustration|artwork|cover\s+art|cover\s+image|logo|visual|diagram|png|mockup)\b`,
	)
	imageGenerationIndirectRE = regexp.MustCompile(
		`\b(?:let'?s|can\s+(?:we|i)|could\s+(?:we|i)|may\s+i|i\s+want\s+to|i'?d\s+like\s+to)\s+` +
			`(?:see|view|preview)\b.{0,120}\b(?:image|picture|illustration|artwork|cover\s+art|cover\s+image|logo|visual|diagram|png|mockup)\b|` +
			`\b(?:see|show|preview)\b.{0,80}\b(?:sample|mockup)\b.{0,80}\b(?:image|picture|illustration|artwork|cover\s+art|cover\s+image|logo|visual|diagram|png)\b`,
	)
	imageGenerationNegationRE = regexp.MustCompile(
		`\b(?:do\s+not|don'?t|never|without|no\s+need\s+to|can\s+you\s+not)\s+.{0,24}` +
			`\b(?:make|generate|create|draw|show|render|design|illustrate|mock\s*up)\b|` +
			`\b(?:you|we|it)\s+(?:can\s+not|cannot|can'?t)\s+.{0,16}` +
			`\b(?:make|generate|create|draw|show|render|design|illustrate|mock\s*up)\b`,
	)
	imageCompanionTextRE = regexp.MustCompile(
		`\b(?:outline|synopsis|summary|caption|description|copy|draft|chapter|text)\b.{0,120}\band\b.{0,120}` +
			`\b(?:image|picture|illustration|artwork|cover\s+art|cover\s+image|logo|visual|diagram|png|mockup)\b|` +
			`\b(?:image|picture|illustration|artwork|cover\s+art|cover\s+image|logo|visual|diagram|png|mockup)\b.{0,120}\band\b.{0,120}` +
			`\b(?:outline|synopsis|summary|caption|description|copy|draft|chapter|text)\b`,
	)
)

// UserRequestsGeneratedImage is a deprecated phrase-matching heuristic. Routing now
// trusts the stamped TurnDecision (Action == ActionImage) instead of natural-language
// phrase matching — see messageSuppressesImageGeneration / tryHubImageGenerationShortcut.
//
// Deprecated: always returns false. Do not add new call sites.
func UserRequestsGeneratedImage(content string) bool {
	return false
}

// UserRequestsImageWithCompanionText reports mixed turns that need both an
// image action and a substantive text deliverable, such as an outline.
func UserRequestsImageWithCompanionText(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	return UserRequestsGeneratedImage(c) && imageCompanionTextRE.MatchString(c)
}

// ImagePromptFromMessage strips mentions and returns a prompt suitable for hub image generation.
func ImagePromptFromMessage(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "A clear technical diagram."
	}
	// Drop leading @mentions
	for strings.HasPrefix(content, "@") {
		if i := strings.IndexByte(content, ' '); i > 0 {
			content = strings.TrimSpace(content[i+1:])
		} else {
			break
		}
	}
	if len(content) > 900 {
		content = content[:900]
	}
	return content
}

// MaybePostHubGeneratedImageForCLI posts a hub-generated image when the user asked for one
// and the CLI response did not attach a local file.
func (a *Agent) MaybePostHubGeneratedImageForCLI(msg *protocol.Message, responseHasImage bool) {
	if !a.isCLIAgent() || responseHasImage || msg == nil || a.Hub == nil {
		return
	}
	explicitImageIntent := UserRequestsGeneratedImage(msg.Content)
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		explicitImageIntent = decision.Action == semantic.ActionImage
	}
	if !a.Hub.ImageGenerationEnabled() || !explicitImageIntent {
		return
	}
	prompt := ImagePromptFromMessage(msg.Content)
	if err := a.Hub.GenerateAndPostImage(context.Background(), msg.Channel, a.Info, prompt, ""); err != nil {
		log.Printf("[%s] hub image fallback: %v", a.Info.Name, err)
	}
}
