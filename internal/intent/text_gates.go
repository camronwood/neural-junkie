package intent

import (
	"os"
	"strings"
	"sync/atomic"
)

// textGatesDisabled disables LooksLike* policy overrides for dual-gate eval.
// When disabled, ResolvePolicy must rely on classifier reason_codes + structural
// features only. Hub workspace-fix preserve may still call LooksLike* directly.
var textGatesDisabled atomic.Bool

func init() {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("NJ_SEMANTIC_TEXT_GATES")))
	if v == "0" || v == "false" || v == "off" || v == "disabled" {
		textGatesDisabled.Store(true)
	}
}

// SetTextGatesDisabled toggles residual phrase policy gates (tests / dual-gate eval).
func SetTextGatesDisabled(disabled bool) {
	textGatesDisabled.Store(disabled)
}

// TextGatesEnabled reports whether LooksLike* policy overrides may fire.
func TextGatesEnabled() bool {
	return !textGatesDisabled.Load()
}

func gateText(fn func(string) bool, text string) bool {
	if !TextGatesEnabled() || fn == nil {
		return false
	}
	return fn(text)
}
