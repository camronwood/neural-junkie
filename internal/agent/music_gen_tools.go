package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const generateMusicToolName = "generate_music"
const extractStemsToolName = "extract_stems"

var generateMusicToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "style_tags": {
      "type": "string",
      "description": "ACE-Step caption: genre, mood, BPM, key instruments, vocal style, production era (comma-separated tags)"
    },
    "lyrics": {
      "type": "string",
      "description": "Song lyrics with [Verse]/[Chorus]/[Bridge] markers, or [Instrumental] for no vocals"
    },
    "duration_sec": {
      "type": "integer",
      "description": "Target length in seconds (10-240, default 30; shorter clips often sound cleaner)"
    },
    "instrumental": {
      "type": "boolean",
      "description": "When true, generate instrumental track without vocals"
    },
    "seed": {
      "type": "integer",
      "description": "Optional random seed for reproducible output (-1 or omit for random)"
    },
    "export_stems": {
      "type": "boolean",
      "description": "When true, also extract stems (requires SFT variant)"
    },
    "stem_tracks": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Stem tracks to export when export_stems is true (e.g. vocals, drums, bass)"
    }
  },
  "required": ["style_tags"]
}`)

var extractStemsToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "audio_path": {
      "type": "string",
      "description": "Absolute or workspace path to the mixed WAV file"
    },
    "tracks": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Tracks to extract (e.g. vocals, drums, bass). Default: vocals and drums"
    }
  },
  "required": ["audio_path"]
}`)

func generateMusicToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        generateMusicToolName,
		Description: "Generate a song or instrumental from style tags and optional lyrics. Posts audio to the current channel. Requires the Music creation pack and ACE-Step sidecar.",
		InputSchema: generateMusicToolSchema,
	}
}

func extractStemsToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        extractStemsToolName,
		Description: "Extract vocals, drums, bass, or other stems from a mixed audio file for DAW handoff. Requires SFT variant. Posts stems to the channel.",
		InputSchema: extractStemsToolSchema,
	}
}

func (a *Agent) musicGenerationToolsEnabledForMessage(msg *protocol.Message) bool {
	if a.Hub == nil || !a.Hub.MusicGenerationEnabled() {
		return false
	}
	if !agentTypeSupportsHubMusicGen(a.Info.Type) {
		return false
	}
	if messageSuppressesImageGeneration(msg) {
		return false
	}
	return true
}

func agentTypeSupportsHubMusicGen(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeMusic, protocol.AgentTypeAssistant:
		return true
	default:
		return false
	}
}

// tryHubMusicGenerationShortcut posts hub-generated audio when the user asked for a song.
func (a *Agent) tryHubMusicGenerationShortcut(ctx context.Context, msg *protocol.Message) (string, bool) {
	if msg == nil || a.Hub == nil || protocol.IsGeneratedAudioDelivery(msg) || !UserRequestsGeneratedMusic(msg.Content) {
		return "", false
	}
	if !a.musicGenerationToolsEnabledForMessage(msg) {
		if !agentTypeSupportsHubMusicGen(a.Info.Type) {
			return "", false
		}
		if a.Hub == nil || a.Hub.MusicGenerationEnabled() {
			return "", false
		}
		reason := "Music generation isn't available — open the **Domain packs** sidebar and enable **Music creation**."
		if hr, ok := a.Hub.(interface{ MusicGenerationUnavailableReason() string }); ok {
			if r := strings.TrimSpace(hr.MusicGenerationUnavailableReason()); r != "" {
				reason = r
			}
		}
		return reason, true
	}
	style := MusicStyleTagsFromMessage(msg.Content)
	if style == "" {
		style = DefaultMusicStyleTags()
	}
	req := MusicGenerateRequest{
		StyleTags:   style,
		DurationSec: 30,
	}
	if musicRequestWantsVocals(msg.Content) {
		req.Lyrics = "[Verse]\nStarting something new tonight\nFinding rhythm in the light\n\n[Chorus]\nSing it loud, sing it clear\nThis is our song right here"
	} else {
		req.Instrumental = true
		req.Lyrics = "[Instrumental]"
	}
	if err := a.generateAndPostMusicWithProgress(ctx, msg, StreamMessageIDFromContext(ctx), req, true); err != nil {
		return fmt.Sprintf("I couldn't generate that song: %v", err), true
	}
	return "Done — I've posted the generated song to the channel.", true
}

func appendMusicGenerationPrompt(system *strings.Builder) {
	system.WriteString("MUSIC GENERATION (ACE-Step):\n")
	system.WriteString("When the user asks you to compose, generate, or produce a song or instrumental, call generate_music with detailed style_tags and lyrics.\n")
	system.WriteString("style_tags must read like an ACE-Step caption: genre, subgenre, mood, tempo (BPM), key instruments, vocal type, production era — e.g. \"upbeat indie pop, 118 bpm, female vocal, acoustic guitar, bright drums, 2010s radio\".\n")
	system.WriteString("For vocals, use section markers in lyrics: [Verse], [Chorus], [Bridge], [Outro]. Keep lines rhythmic (roughly 4–8 syllables per line).\n")
	system.WriteString("For instrumentals set instrumental=true and lyrics=\"[Instrumental]\"; emphasize instrumentation and mood in style_tags.\n")
	system.WriteString("Draft or refine lyrics in chat first when helpful; default duration_sec=30 unless the user asks longer. Offer to iterate with a new seed or revised tags.\n")
	system.WriteString("For DAW handoff, use extract_stems with audio_path (requires SFT variant) or generate_music with export_stems=true.\n")
	system.WriteString("When a music workbench project is open, prefer updating project.nj-music.json sections before generating.\n")
	system.WriteString("After success, briefly confirm — audio is posted automatically.\n\n")
}

func (a *Agent) generateAndPostMusicWithProgress(
	ctx context.Context,
	msg *protocol.Message,
	streamMsgID string,
	req MusicGenerateRequest,
	broadcastToolStart bool,
) error {
	a.sendThinkingActivity(msg, protocol.ThinkingActivityGeneratingMusic, musicGenToolPreview(req.StyleTags))
	defer a.clearThinkingActivity(msg)

	if broadcastToolStart && streamMsgID != "" {
		a.broadcastToolStep(ctx, msg, streamMsgID, ai.ToolStepEvent{
			Kind:    "start",
			Name:    generateMusicToolName,
			Preview: musicGenToolPreview(req.StyleTags),
		})
	}

	err := a.Hub.GenerateAndPostMusic(ctx, msg.Channel, a.Info, req)

	if broadcastToolStart && streamMsgID != "" {
		ev := ai.ToolStepEvent{Kind: "done", Name: generateMusicToolName, Preview: "Song ready"}
		if err != nil {
			ev.Kind = "error"
			ev.Preview = err.Error()
		}
		a.broadcastToolStep(ctx, msg, streamMsgID, ev)
	}
	return err
}

func musicGenToolPreview(style string) string {
	style = strings.TrimSpace(style)
	if style == "" {
		return "Generating music…"
	}
	const max = 80
	if len(style) <= max {
		return "Generating music: " + style
	}
	return "Generating music: " + style[:max] + "…"
}

func (a *Agent) executeGenerateMusicTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if messageSuppressesImageGeneration(msg) {
		return "", fmt.Errorf("generate_music is not available during implementation or code-editing sessions")
	}
	var args struct {
		StyleTags    string   `json:"style_tags"`
		Lyrics       string   `json:"lyrics"`
		DurationSec  int      `json:"duration_sec"`
		Instrumental bool     `json:"instrumental"`
		Seed         int      `json:"seed"`
		ExportStems  bool     `json:"export_stems"`
		StemTracks   []string `json:"stem_tracks"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid generate_music input: %w", err)
		}
	}
	args.StyleTags = strings.TrimSpace(args.StyleTags)
	if args.StyleTags == "" {
		return "", fmt.Errorf("generate_music requires non-empty style_tags")
	}
	req := MusicGenerateRequest{
		StyleTags:    args.StyleTags,
		Lyrics:       strings.TrimSpace(args.Lyrics),
		DurationSec:  args.DurationSec,
		Instrumental: args.Instrumental,
		Seed:         args.Seed,
		ExportStems:  args.ExportStems,
		StemTracks:   args.StemTracks,
	}
	if err := a.generateAndPostMusicWithProgress(ctx, msg, StreamMessageIDFromContext(ctx), req, false); err != nil {
		return "", err
	}
	return "Song generated and posted to the channel.", nil
}

func (a *Agent) executeExtractStemsTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if messageSuppressesImageGeneration(msg) {
		return "", fmt.Errorf("extract_stems is not available during implementation or code-editing sessions")
	}
	var args struct {
		AudioPath string   `json:"audio_path"`
		Tracks    []string `json:"tracks"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid extract_stems input: %w", err)
		}
	}
	args.AudioPath = strings.TrimSpace(args.AudioPath)
	if args.AudioPath == "" {
		return "", fmt.Errorf("extract_stems requires audio_path")
	}
	req := MusicExtractRequest{
		AudioPath: args.AudioPath,
		Tracks:    args.Tracks,
	}
	if err := a.Hub.ExtractAndPostMusicStems(ctx, msg.Channel, a.Info, req); err != nil {
		return "", err
	}
	return "Stems extracted and posted to the channel.", nil
}
