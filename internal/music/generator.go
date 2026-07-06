package music

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StemResult is one extracted or generated stem track.
type StemResult struct {
	Track string `json:"track"`
	Path  string `json:"path,omitempty"`
	Mime  string `json:"mime"`
	Data  string `json:"data"`
}

// Result is the full sidecar generation response.
type Result struct {
	Mime          string       `json:"mime"`
	Data          string       `json:"data"`
	Path          string       `json:"path,omitempty"`
	GenerationID  string       `json:"generation_id,omitempty"`
	Stems         []StemResult `json:"stems,omitempty"`
}

// Request is passed to the music generation backend (ACE-Step sidecar).
type Request struct {
	StyleTags      string  `json:"style_tags"`
	Lyrics         string  `json:"lyrics"`
	DurationSec    int     `json:"duration_sec"`
	Instrumental   bool    `json:"instrumental"`
	Seed           int     `json:"seed,omitempty"`
	InferenceSteps int     `json:"inference_steps,omitempty"`
	GuidanceScale  float64 `json:"guidance_scale,omitempty"`
	InferMethod    string  `json:"infer_method,omitempty"`
	ExportStems    bool    `json:"export_stems,omitempty"`
	StemTracks     []string `json:"stem_tracks,omitempty"`
}

// Generator produces audio from style tags and lyrics.
type Generator interface {
	Generate(ctx context.Context, req Request) (Result, error)
}

// Default is wired by the hub at startup (pack sidecar).
var Default Generator

// SidecarBaseURL resolves the music pack sidecar base URL when wired by the hub server.
var SidecarBaseURL func() string

// ResolveGenerator returns the configured generator or a lazy sidecar client when SidecarBaseURL is set.
func ResolveGenerator() Generator {
	if Default != nil {
		return Default
	}
	if SidecarBaseURL != nil {
		return NewSidecarGenerator(SidecarBaseURL)
	}
	return nil
}

// SidecarGenerator calls POST /api/music/* on a pack sidecar.
type SidecarGenerator struct {
	BaseURL func() string
	Client  *http.Client
}

// NewSidecarGenerator returns a generator that posts to a dynamic sidecar base URL.
func NewSidecarGenerator(baseURL func() string) *SidecarGenerator {
	return &SidecarGenerator{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 30 * time.Minute},
	}
}

type sidecarResponse struct {
	Mime         string       `json:"mime"`
	Data         string       `json:"data"`
	Path         string       `json:"path"`
	GenerationID string       `json:"generation_id"`
	Stems        []StemResult `json:"stems"`
	Error        string       `json:"error"`
}

// Generate invokes the sidecar music API.
func (g *SidecarGenerator) Generate(ctx context.Context, req Request) (Result, error) {
	if g == nil || g.BaseURL == nil {
		return Result{}, fmt.Errorf("music sidecar generator not configured")
	}
	base := strings.TrimRight(strings.TrimSpace(g.BaseURL()), "/")
	if base == "" {
		return Result{}, fmt.Errorf("music sidecar not running (enable Music creation pack)")
	}
	out, err := g.postJSON(ctx, base+"/api/music/generate", req)
	if err != nil {
		return Result{}, err
	}
	if req.ExportStems && len(out.Stems) == 0 && out.Path != "" {
		tracks := req.StemTracks
		if len(tracks) == 0 {
			tracks = []string{"vocals", "drums"}
		}
		extractBody := map[string]interface{}{
			"audio_path": out.Path,
			"tracks":     tracks,
		}
		extractOut, extractErr := g.postJSON(ctx, base+"/api/music/extract", extractBody)
		if extractErr == nil && len(extractOut.Stems) > 0 {
			out.Stems = extractOut.Stems
		}
	}
	return out, nil
}

// ExtractStems calls POST /api/music/extract.
func (g *SidecarGenerator) ExtractStems(ctx context.Context, audioPath string, tracks []string) (Result, error) {
	if g == nil || g.BaseURL == nil {
		return Result{}, fmt.Errorf("music sidecar generator not configured")
	}
	base := strings.TrimRight(strings.TrimSpace(g.BaseURL()), "/")
	if base == "" {
		return Result{}, fmt.Errorf("music sidecar not running")
	}
	if len(tracks) == 0 {
		tracks = []string{"vocals", "drums"}
	}
	return g.postJSON(ctx, base+"/api/music/extract", map[string]interface{}{
		"audio_path": audioPath,
		"tracks":     tracks,
	})
}

func (g *SidecarGenerator) postJSON(ctx context.Context, url string, body interface{}) (Result, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return Result{}, fmt.Errorf("music sidecar: %s", msg)
	}
	var out sidecarResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return Result{}, fmt.Errorf("decode music response: %w", err)
	}
	if errMsg := strings.TrimSpace(out.Error); errMsg != "" {
		return Result{}, fmt.Errorf("%s", errMsg)
	}
	mime := strings.TrimSpace(out.Mime)
	data := strings.TrimSpace(out.Data)
	if mime == "" || data == "" {
		return Result{}, fmt.Errorf("music sidecar returned empty audio")
	}
	return Result{
		Mime:         mime,
		Data:         data,
		Path:         out.Path,
		GenerationID: out.GenerationID,
		Stems:        out.Stems,
	}, nil
}
