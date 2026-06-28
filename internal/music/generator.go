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
}

// Generator produces audio from style tags and lyrics.
type Generator interface {
	Generate(ctx context.Context, req Request) (mime string, dataB64 string, err error)
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

// SidecarGenerator calls POST /api/music/generate on a pack sidecar.
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

type generateResponse struct {
	Mime string `json:"mime"`
	Data string `json:"data"`
	Error string `json:"error"`
}

// Generate invokes the sidecar music API.
func (g *SidecarGenerator) Generate(ctx context.Context, req Request) (string, string, error) {
	if g == nil || g.BaseURL == nil {
		return "", "", fmt.Errorf("music sidecar generator not configured")
	}
	base := strings.TrimRight(strings.TrimSpace(g.BaseURL()), "/")
	if base == "" {
		return "", "", fmt.Errorf("music sidecar not running (enable Music creation pack)")
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return "", "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/music/generate", bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return "", "", fmt.Errorf("music sidecar: %s", msg)
	}
	var out generateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", fmt.Errorf("decode music response: %w", err)
	}
	if errMsg := strings.TrimSpace(out.Error); errMsg != "" {
		return "", "", fmt.Errorf("%s", errMsg)
	}
	mime := strings.TrimSpace(out.Mime)
	data := strings.TrimSpace(out.Data)
	if mime == "" || data == "" {
		return "", "", fmt.Errorf("music sidecar returned empty audio")
	}
	return mime, data, nil
}
