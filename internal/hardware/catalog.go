package hardware

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ollama"
)

var sizeHintRe = regexp.MustCompile(`(?i)([\d.]+)\s*GB`)

// ModelLookup is catalog metadata plus derived estimates for UI and docs.
type ModelLookup struct {
	Name             string  `json:"name"`
	Title            string  `json:"title,omitempty"`
	SizeHint         string  `json:"size_hint,omitempty"`
	EstimatedDiskGB  float64 `json:"estimated_disk_gb,omitempty"`
	EstimatedRAMGB   int     `json:"estimated_ram_gb,omitempty"`
}

// ParseSizeHintGB extracts the leading GB value from strings like "~9 GB" or "~4.5 GB (Q4)".
func ParseSizeHintGB(sizeHint string) (float64, bool) {
	sizeHint = strings.TrimSpace(sizeHint)
	if sizeHint == "" {
		return 0, false
	}
	m := sizeHintRe.FindStringSubmatch(sizeHint)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// EstimatedRAMGBFromModelSize returns ceil(modelGB * 1.25 + 4) for OS/hub/runtime headroom.
func EstimatedRAMGBFromModelSize(modelGB float64) int {
	if modelGB <= 0 {
		return 0
	}
	return int(math.Ceil(modelGB*1.25 + 4))
}

// LookupModel finds a catalog row by Ollama tag (case-sensitive name match).
func LookupModel(name string) (*ModelLookup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	models, err := ollama.Library()
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		if m.Name != name {
			continue
		}
		out := &ModelLookup{
			Name:  m.Name,
			Title: m.Title,
		}
		if m.SizeHint != "" {
			out.SizeHint = m.SizeHint
			if gb, ok := ParseSizeHintGB(m.SizeHint); ok {
				out.EstimatedDiskGB = gb
				out.EstimatedRAMGB = EstimatedRAMGBFromModelSize(gb)
			}
		}
		return out, nil
	}
	return nil, nil
}
