package contextcompress

import (
	"context"
	"strings"
	"sync/atomic"
)

type compressContextKey struct{}
type retrieveCountKey struct{}

// CompressContext holds channel and tool-call ids for compression.
type CompressContext struct {
	ChannelID string
	CallID    string
}

// WithCompressContext attaches channel and tool-call id for compression.
func WithCompressContext(ctx context.Context, channelID, callID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, compressContextKey{}, CompressContext{
		ChannelID: strings.TrimSpace(channelID),
		CallID:    strings.TrimSpace(callID),
	})
}

// CompressContextFrom returns channel/call ids when present.
func CompressContextFrom(ctx context.Context) (channelID, callID string) {
	if ctx == nil {
		return "", ""
	}
	if v, ok := ctx.Value(compressContextKey{}).(CompressContext); ok {
		return v.ChannelID, v.CallID
	}
	return "", ""
}

// WithRetrieveBudget attaches a per-turn retrieve counter (max MaxRetrievePerTurn).
func WithRetrieveBudget(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	var n int32
	return context.WithValue(ctx, retrieveCountKey{}, &n)
}

type retrieveBudgetKey struct{}

// WithAgentRetrieveBudget raises per-turn nj_retrieve_context cap for agent-runtime loops.
func WithAgentRetrieveBudget(ctx context.Context, max int) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if max <= 0 {
		max = maxRetrievePerTurn
	}
	return context.WithValue(ctx, retrieveBudgetKey{}, max)
}

func retrieveBudgetFromContext(ctx context.Context) int {
	if ctx == nil {
		return maxRetrievePerTurn
	}
	if v, ok := ctx.Value(retrieveBudgetKey{}).(int); ok && v > 0 {
		return v
	}
	return maxRetrievePerTurn
}

// TryConsumeRetrieve returns false when the per-turn retrieve budget is exhausted.
func TryConsumeRetrieve(ctx context.Context) bool {
	max := retrieveBudgetFromContext(ctx)
	if ctx == nil {
		return true
	}
	p, ok := ctx.Value(retrieveCountKey{}).(*int32)
	if !ok || p == nil {
		return true
	}
	if atomic.LoadInt32(p) >= int32(max) {
		return false
	}
	atomic.AddInt32(p, 1)
	return true
}

// MaxRetrievePerTurn is the per-turn cap on nj_retrieve_context calls.
const MaxRetrievePerTurn = maxRetrievePerTurn
