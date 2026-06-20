package contextcompress

import (
	"github.com/camronwood/neural-junkie/internal/ai"
)

// RuntimeOptions returns compression options from hub performance config.
func RuntimeOptions() Options {
	return OptionsFromPerformance(ai.PerformanceConfig())
}
