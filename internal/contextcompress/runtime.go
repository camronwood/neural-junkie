package contextcompress

import (
	"github.com/camronwood/neural-junkie/internal/ai"
)

// RuntimeOptions returns compression options from hub performance config.
func RuntimeOptions() Options {
	return OptionsFromPerformance(ai.PerformanceConfig())
}

// RuntimeOptionsForAgent returns elevated compression limits for agent-runtime v2.
func RuntimeOptionsForAgent() Options {
	o := OptionsFromPerformance(ai.PerformanceConfig())
	p := ai.PerformanceConfig()
	o.MaxToolBytes = p.AgentRuntimeMaxToolBytesOrDefault()
	return o.normalized()
}
