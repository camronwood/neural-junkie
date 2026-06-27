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

var generateMusicToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "style_tags": {
      "type": "string",
      "description": "Genre, mood, tempo, and instrumentation (ACE-Step caption/tags)"
    },
    "lyrics": {
      "type": "string",
      "description": "Song lyrics with optional section markers, or [Instrumental] for no vocals"
    },
    "duration_sec": {
      "type": "integer",
      "description": "Target length in seconds (10-240, default 30)"
    },
    "instrumental": {
      "type": "boolean",
      "description": "When true, generate instrumental track without vocals"
    }
  },
  "required": ["style_tags"]
}`)

func generateMusicToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        generateMusicToolName,
		Description: "Generate a song or instrumental from style tags and optional lyrics. Posts audio to the current channel. Requires the Music creation pack and ACE-Step sidecar.",
		InputSchema: generateMusicToolSchema,
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

func appendMusicGenerationPrompt(system *strings.Builder) {
	system.WriteString("MUSIC GENERATION:\n")
	system.WriteString("When the user asks you to compose, generate, or produce a song or instrumental, call generate_music with detailed style_tags and lyrics.\n")
	system.WriteString("Draft or refine lyrics in chat first when helpful; then call the tool. After success, briefly confirm — audio is posted automatically.\n\n")
}

func (a *Agent) generateAndPostMusicWithProgress(
	ctx context.Context,
	msg *protocol.Message,
	streamMsgID string,
	req MusicGenerateRequest,
	broadcastToolStart bool,
) error {
	a.sendThinkingActivity(msg, protocol.ThinkingActivityGeneratingMusic, musicGenToolPreview(req.StyleTags))
	defer a.sendThinkingActivity(msg, "")

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
		StyleTags    string `json:"style_tags"`
		Lyrics       string `json:"lyrics"`
		DurationSec  int    `json:"duration_sec"`
		Instrumental bool   `json:"instrumental"`
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
	}
	if err := a.generateAndPostMusicWithProgress(ctx, msg, StreamMessageIDFromContext(ctx), req, false); err != nil {
		return "", err
	}
	return "Song generated and posted to the channel.", nil
}
