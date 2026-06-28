package agent

import "context"

type implRoutingHintsKey struct{}

// ImplementationRoutingHints carries session-state signals for reliable-tier routing.
type ImplementationRoutingHints struct {
	RepairAttempts int
	VerifyFailed   bool
	BootFixIntent  bool
}

// ContextWithImplementationRoutingHints attaches repair/boot-fix routing hints.
func ContextWithImplementationRoutingHints(ctx context.Context, hints ImplementationRoutingHints) context.Context {
	if ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, implRoutingHintsKey{}, hints)
}

// ImplementationRoutingHintsFromContext returns hints when present.
func ImplementationRoutingHintsFromContext(ctx context.Context) ImplementationRoutingHints {
	if ctx == nil {
		return ImplementationRoutingHints{}
	}
	h, _ := ctx.Value(implRoutingHintsKey{}).(ImplementationRoutingHints)
	return h
}
