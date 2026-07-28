package hub

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
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
	return true
}

// MusicGenerationUnavailableReason explains why generation is disabled for user-facing errors.
func (h *Hub) MusicGenerationUnavailableReason() string {
	if h == nil || h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return "Music generation is unavailable — hub configuration is not loaded."
	}
	cfg := h.commandHandler.appConfig
	const sidebar = "the **Domain packs** button in the sidebar"
	if !cfg.IsPackInstalled(config.PackMusicCreation) {
		return "Install the **Music creation** pack from " + sidebar + " (Store tab)."
	}
	if !cfg.IsPackEnabled(config.PackMusicCreation) {
		return "Enable **Music creation** from " + sidebar + " (Store tab), then try again."
	}
	if !cfg.AnyPackCapability(capMusicGeneration) {
		return "Music generation capability is inactive — toggle **Music creation** off and on in " + sidebar + ", then restart the hub."
	}
	if music.ResolveGenerator() == nil {
		return "Music generator is not wired — restart the hub."
	}
	st := h.ACEStepStatus()
	if st.DemoMode || st.Ready {
		return "Music generation hit an unexpected error — try again or restart the hub."
	}
	if st.Installing {
		return "ACE-Step is still installing — open " + sidebar + " → **Tools** tab to watch progress."
	}
	if msg := strings.TrimSpace(st.LastError); msg != "" {
		return "ACE-Step is not ready: " + msg + " Open " + sidebar + " → **Tools** to fix it."
	}
	return "ACE-Step weights are not installed for the selected model — open " + sidebar + " → **Tools** and install weights."
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
	if ag := h.FindLiveAgentByDisplayName("Assistant", protocol.AgentTypeAssistant); ag != nil {
		return *ag
	}
	if ag := h.FindLiveAgentByDisplayName("MusicExpert", protocol.AgentTypeMusic); ag != nil {
		return *ag
	}
	return protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral}
}

// GenerateAndPostMusic generates audio and posts it to a channel.
func (h *Hub) GenerateAndPostMusic(ctx context.Context, channel string, from protocol.AgentInfo, req agent.MusicGenerateRequest) error {
	if !h.MusicGenerationAvailable() {
		return fmt.Errorf("music generation disabled (install and enable the Music creation pack)")
	}
	gen := music.ResolveGenerator()
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
	if h.commandHandler != nil && h.commandHandler.appConfig != nil {
		def := h.commandHandler.appConfig.MusicMCPSettings().Normalized()
		if req.InferenceSteps <= 0 {
			req.InferenceSteps = def.InferenceSteps
		}
		if req.GuidanceScale <= 0 {
			req.GuidanceScale = def.GuidanceScale
		}
		if strings.TrimSpace(req.InferMethod) == "" {
			req.InferMethod = def.InferMethod
		}
		if req.Seed == 0 && def.DefaultSeed != 0 {
			req.Seed = def.DefaultSeed
		}
	}

	result, err := gen.Generate(ctx, music.Request{
		StyleTags:      req.StyleTags,
		Lyrics:         req.Lyrics,
		DurationSec:    req.DurationSec,
		Instrumental:   req.Instrumental,
		Seed:           req.Seed,
		InferenceSteps: req.InferenceSteps,
		GuidanceScale:  req.GuidanceScale,
		InferMethod:    req.InferMethod,
		ExportStems:    req.ExportStems,
		StemTracks:     req.StemTracks,
	})
	if err != nil {
		return err
	}

	out := protocol.NewMessage(protocol.MessageTypeChat, channel, from, protocol.GeneratedAudioDeliveryContent)
	meta := map[string]interface{}{
		"mime":       result.Mime,
		"data":       result.Data,
		"style_tags": req.StyleTags,
	}
	if req.Lyrics != "" {
		meta["lyrics"] = req.Lyrics
	}
	if result.GenerationID != "" {
		meta["generation_id"] = result.GenerationID
	}
	if len(result.Stems) > 0 {
		stemsMeta := make([]map[string]interface{}, 0, len(result.Stems))
		for _, stem := range result.Stems {
			entry := map[string]interface{}{
				"track": stem.Track,
				"mime":  stem.Mime,
				"data":  stem.Data,
			}
			if stem.Path != "" {
				entry["path"] = stem.Path
			}
			stemsMeta = append(stemsMeta, entry)
		}
		meta["stems"] = stemsMeta
	}
	if path, err := saveGeneratedAudioFile(out.ID, result.Mime, result.Data); err != nil {
		log.Printf("[hub] generated audio file save failed (inline only): %v", err)
	} else {
		meta["path"] = path
	}
	out.Metadata["generated_audio"] = meta
	return h.SendMessage(out)
}

// ExtractAndPostMusicStems extracts stems from an audio file and posts to the channel.
func (h *Hub) ExtractAndPostMusicStems(ctx context.Context, channel string, from protocol.AgentInfo, req agent.MusicExtractRequest) error {
	if !h.MusicGenerationAvailable() {
		return fmt.Errorf("music generation disabled (install and enable the Music creation pack)")
	}
	gen := music.ResolveGenerator()
	sg, ok := gen.(*music.SidecarGenerator)
	if !ok {
		return fmt.Errorf("stem extraction requires the music pack sidecar")
	}
	req.AudioPath = strings.TrimSpace(req.AudioPath)
	if req.AudioPath == "" {
		return fmt.Errorf("audio_path required")
	}
	if len(req.Tracks) == 0 {
		req.Tracks = []string{"vocals", "drums"}
	}
	result, err := sg.ExtractStems(ctx, req.AudioPath, req.Tracks)
	if err != nil {
		return err
	}
	out := protocol.NewMessage(protocol.MessageTypeChat, channel, from, protocol.GeneratedAudioDeliveryContent+" (stems)")
	meta := map[string]interface{}{
		"mime": result.Mime,
		"data": result.Data,
		"path": req.AudioPath,
	}
	if result.GenerationID != "" {
		meta["generation_id"] = result.GenerationID
	}
	if len(result.Stems) > 0 {
		stemsMeta := make([]map[string]interface{}, 0, len(result.Stems))
		for _, stem := range result.Stems {
			entry := map[string]interface{}{
				"track": stem.Track,
				"mime":  stem.Mime,
				"data":  stem.Data,
			}
			if stem.Path != "" {
				entry["path"] = stem.Path
			}
			stemsMeta = append(stemsMeta, entry)
		}
		meta["stems"] = stemsMeta
	}
	out.Metadata["generated_audio"] = meta
	return h.SendMessage(out)
}
