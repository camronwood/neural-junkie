package hub

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/music"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubMusicGen struct{}

func (stubMusicGen) Generate(_ context.Context, req music.Request) (music.Result, error) {
	return music.Result{Mime: "audio/wav", Data: "UklGRg=="}, nil
}

func TestMusicGenerationAvailableRequiresPackCapabilityOnly(t *testing.T) {
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackMusicCreation)
	cfg.Packs.Enabled[config.PackMusicCreation] = true

	oldDefault := music.Default
	oldSidecar := music.SidecarBaseURL
	music.Default = nil
	music.SidecarBaseURL = nil
	t.Cleanup(func() {
		music.Default = oldDefault
		music.SidecarBaseURL = oldSidecar
	})

	h := NewHub()
	h.commandHandler = &CommandHandler{hub: h, appConfig: cfg}
	if !h.MusicGenerationAvailable() {
		t.Fatal("expected music generation available when pack capability is enabled")
	}
}

func TestGenerateAndPostMusic(t *testing.T) {
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackMusicCreation)
	cfg.Packs.Enabled[config.PackMusicCreation] = true

	old := music.Default
	music.Default = stubMusicGen{}
	t.Cleanup(func() { music.Default = old })

	h := NewHub()
	h.commandHandler = &CommandHandler{hub: h, appConfig: cfg}

	from := protocol.AgentInfo{Name: "MusicExpert", Type: protocol.AgentTypeMusic}
	if err := h.GenerateAndPostMusic(context.Background(), "general", from, agent.MusicGenerateRequest{
		StyleTags: "lo-fi chill",
		Lyrics:    "[Instrumental]",
	}); err != nil {
		t.Fatal(err)
	}
	msgs, err := h.GetMessages("general", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected message")
	}
	last := msgs[len(msgs)-1]
	if !protocol.IsGeneratedAudioDelivery(last) {
		t.Fatalf("expected audio delivery, got %q", last.Content)
	}
}
