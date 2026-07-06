package train

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

// SidecarRunRequest is POST /api/lora/sidecar/run on the pack sidecar.
type SidecarRunRequest struct {
	Dataset       string  `json:"dataset"`
	OutputDir     string  `json:"output_dir"`
	BaseModel     string  `json:"base_model"`
	Rank          int     `json:"rank,omitempty"`
	Epochs        int     `json:"epochs,omitempty"`
	LearningRate  float64 `json:"learning_rate,omitempty"`
	MaxSeqLen     int     `json:"max_seq_len,omitempty"`
	Backend       string  `json:"backend,omitempty"`
	ResumeAdapter string  `json:"resume_adapter,omitempty"`
}

// SidecarRunner runs training subprocess in the pack sidecar.
type SidecarRunner func(ctx context.Context, baseURL string, req SidecarRunRequest) error

// RunViaSidecar POSTs a training job to the specialist-tuning sidecar.
func RunViaSidecar(ctx context.Context, baseURL string, req SidecarRunRequest) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("sidecar base URL required")
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/lora/sidecar/run", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 6 * time.Hour}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sidecar train: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

// SidecarReady GETs /api/lora/sidecar/status from the pack sidecar.
func SidecarReady(ctx context.Context, baseURL string) bool {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/lora/sidecar/status", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var payload struct {
		Ready bool `json:"ready"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false
	}
	return payload.Ready
}
