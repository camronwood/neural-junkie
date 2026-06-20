package contextcompress

import (
	"github.com/camronwood/neural-junkie/internal/config"
)

const (
	defaultMaxToolBytes   = 12000
	defaultMaxEntries     = 500
	defaultTTLMinutes     = 60
	defaultListTopN       = 40
	defaultLogTailLines   = 80
	defaultReadHeadLines  = 40
	defaultReadTailLines  = 20
	maxRetrievePerTurn    = 3
)

// Options configures compression behavior.
type Options struct {
	Enabled      bool
	MaxToolBytes int
	MaxEntries   int
	TTLMinutes   int
	ListTopN     int
}

// DefaultOptions returns enabled compression with defaults.
func DefaultOptions() Options {
	return Options{
		Enabled:      true,
		MaxToolBytes: defaultMaxToolBytes,
		MaxEntries:   defaultMaxEntries,
		TTLMinutes:   defaultTTLMinutes,
		ListTopN:     defaultListTopN,
	}
}

// OptionsFromPerformance maps hub performance config.
func OptionsFromPerformance(p config.PerformanceConfig) Options {
	o := DefaultOptions()
	if p.ContextCompressEnabled != nil && !*p.ContextCompressEnabled {
		o.Enabled = false
	}
	if p.ContextCompressMaxToolBytes > 0 {
		o.MaxToolBytes = p.ContextCompressMaxToolBytes
	}
	if p.ContextCacheMaxEntries > 0 {
		o.MaxEntries = p.ContextCacheMaxEntries
	}
	if p.ContextCacheTTLMinutes > 0 {
		o.TTLMinutes = p.ContextCacheTTLMinutes
	}
	return o
}

func (o Options) normalized() Options {
	out := o
	if out.MaxToolBytes <= 0 {
		out.MaxToolBytes = defaultMaxToolBytes
	}
	if out.MaxEntries <= 0 {
		out.MaxEntries = defaultMaxEntries
	}
	if out.TTLMinutes <= 0 {
		out.TTLMinutes = defaultTTLMinutes
	}
	if out.ListTopN <= 0 {
		out.ListTopN = defaultListTopN
	}
	return out
}
