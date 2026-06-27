package hub

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/music"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const capMusicGeneration = "music-generation"

// MusicGenerationAvailable reports whether the hub can generate music via the music-creation pack.
func (h *Hub) MusicGenerationAvailable() bool {
	if h == nil || h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return false
	}
	if !h.commandHandler.appConfig.AnyPackCapability(capMusicGeneration) {
		return false
	}
	return music.Default != nil
}

func agentTypeSupportsMusicGeneration(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeMusic, protocol.AgentTypeAssistant:
		return true
	default:
		return false
	}
}

// enrichAgentMusicGeneration sets SupportsMusicGeneration when the music pack is active.
func enrichAgentMusicGeneration(h *Hub, agent *protocol.AgentInfo) {
	if h == nil || agent == nil {
		return
	}
	agent.SupportsMusicGeneration = h.MusicGenerationAvailable() && agentTypeSupportsMusicGeneration(agent.Type)
}

// MusicGenerationEnabled implements agent.HubClient.
func (h *Hub) MusicGenerationEnabled() bool {
	return h.MusicGenerationAvailable()
}

func (h *Hub) resolveMusicPostAgent(channel string) protocol.AgentInfo {
	if h.isChannelDM(channel) {
		if id := h.primaryAgentIDForDM(channel); id != "" {
			if ag, err := h.GetAgent(id); err == nil && ag != nil {
				return *ag
			}
		}
	}
	if agents, err := h.GetChannelAgents(channel); err == nil {
		for _, ag := range agents {
			if ag.SupportsMusicGeneration {
				return ag
			}
		}
		if len(agents) > 0 {
			return agents[0]
		}
	}
	if ag := h.FindLiveAgentByDisplayName("MusicExpert", protocol.AgentTypeMusic); ag != nil {
		return *ag
	}
	if ag := h.FindLiveAgentByDisplayName("Assistant", protocol.AgentTypeAssistant); ag != nil {
		return *ag
	}
	return protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral}
}

// GenerateAndPostMusic generates audio and posts it to a channel.
func (h *Hub) GenerateAndPostMusic(ctx context.Context, channel string, from protocol.AgentInfo, req agent.MusicGenerateRequest) error {
	if !h.MusicGenerationAvailable() {
		return fmt.Errorf("music generation disabled (install and enable the Music creation pack)")
	}
	gen := music.Default
	if gen == nil {
		return fmt.Errorf("music generator not configured")
	}
	req.StyleTags = strings.TrimSpace(req.StyleTags)
	req.Lyrics = strings.TrimSpace(req.Lyrics)
	if req.StyleTags == "" && req.Lyrics == "" {
		return fmt.Errorf("style_tags or lyrics required")
	}
	if req.DurationSec <= 0 {
		req.DurationSec = 30
	}
	if req.DurationSec > 240 {
		req.DurationSec = 240
	}
	if req.Instrumental && req.Lyrics == "" {
		req.Lyrics = "[Instrumental]"
	}

	mime, b64, err := gen.Generate(ctx, music.Request{
		StyleTags:    req.StyleTags,
		Lyrics:       req.Lyrics,
		DurationSec:  req.DurationSec,
		Instrumental: req.Instrumental,
		Seed:         req.Seed,
	})
	if err != nil {
		return err
	}

	out := protocol.NewMessage(protocol.MessageTypeChat, channel, from, protocol.GeneratedAudioDeliveryContent)
	meta := map[string]interface{}{
		"mime":       mime,
		"data":       b64,
		"style_tags": req.StyleTags,
	}
	if req.Lyrics != "" {
		meta["lyrics"] = req.Lyrics
	}
	if path, err := saveGeneratedAudioFile(out.ID, mime, b64); err != nil {
		log.Printf("[hub] generated audio file save failed (inline only): %v", err)
	} else {
		meta["path"] = path
	}
	out.Metadata["generated_audio"] = meta
	return h.SendMessage(out)
}
