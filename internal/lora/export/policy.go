package export

import (
	"github.com/camronwood/neural-junkie/internal/packs"
)

// MinRowsFromPolicy returns the training row threshold from pack lora_policy.
func MinRowsFromPolicy(p packs.LoRAPolicy) int {
	p = p.Resolved()
	if p.SuggestAfterTurns > 0 {
		return p.SuggestAfterTurns
	}
	return MinRows
}

// RefreshDeltaFromPolicy returns incremental refresh delta from pack lora_policy.
func RefreshDeltaFromPolicy(p packs.LoRAPolicy) int {
	p = p.Resolved()
	if p.RefreshAfterDelta > 0 {
		return p.RefreshAfterDelta
	}
	return DefaultRefreshDelta
}
