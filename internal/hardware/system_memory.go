package hardware

import (
	"context"
	"time"

	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
)

// LoadedOllamaModel is a model currently resident in Ollama RAM/VRAM.
type LoadedOllamaModel struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	VRAMBytes int64  `json:"vram_bytes"`
}

// OllamaMemoryStatus summarizes loaded models from a running Ollama server.
type OllamaMemoryStatus struct {
	Running          bool                `json:"running"`
	Endpoint         string              `json:"endpoint"`
	LoadedModels     []LoadedOllamaModel `json:"loaded_models"`
	LoadedBytesTotal int64                `json:"loaded_bytes_total"`
}

// SystemMemorySnapshot is the live memory monitor payload for the hub API.
type SystemMemorySnapshot struct {
	TotalBytes            uint64             `json:"total_bytes"`
	AvailableBytes        uint64             `json:"available_bytes"`
	UsedBytes             uint64             `json:"used_bytes"`
	UsedPercent           float64            `json:"used_percent"`
	Tier                  Tier               `json:"tier"`
	AppMemoryBytes        uint64             `json:"app_memory_bytes,omitempty"`
	WiredMemoryBytes      uint64             `json:"wired_memory_bytes,omitempty"`
	CompressedMemoryBytes uint64             `json:"compressed_memory_bytes,omitempty"`
	Ollama                OllamaMemoryStatus `json:"ollama"`
}

// BuildSystemMemorySnapshot probes system RAM and optional Ollama loaded models.
func BuildSystemMemorySnapshot(ollamaEndpoint string) (SystemMemorySnapshot, error) {
	usage, err := MemoryUsageSnapshot()
	if err != nil {
		return SystemMemorySnapshot{}, err
	}

	endpoint := ollamaEndpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	status := OllamaMemoryStatus{
		Running:      false,
		Endpoint:     endpoint,
		LoadedModels: []LoadedOllamaModel{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mgr := ollamaManager.NewManager(endpoint)
	if mgr.IsServerRunning(ctx) {
		status.Running = true
		if models, err := mgr.RunningModels(ctx); err == nil {
			for _, m := range models {
				status.LoadedModels = append(status.LoadedModels, LoadedOllamaModel{
					Name:      m.Name,
					SizeBytes: m.SizeBytes,
					VRAMBytes: m.VRAMBytes,
				})
				status.LoadedBytesTotal += m.SizeBytes
			}
		}
	}

	return SystemMemorySnapshot{
		TotalBytes:            usage.TotalBytes,
		AvailableBytes:        usage.AvailableBytes,
		UsedBytes:             usage.UsedBytes,
		UsedPercent:           usage.UsedPercent,
		Tier:                  TierForMemoryGB(MemoryGB(usage.TotalBytes)),
		AppMemoryBytes:        usage.AppMemoryBytes,
		WiredMemoryBytes:      usage.WiredMemoryBytes,
		CompressedMemoryBytes: usage.CompressedMemoryBytes,
		Ollama:                status,
	}, nil
}
