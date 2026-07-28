package agent

import "context"

type capabilityHandoffTurnKey struct{}

type capabilityHandoffTurnState struct {
	count int
}

func withCapabilityHandoffTurnState(ctx context.Context) context.Context {
	if capabilityHandoffTurnStateFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, capabilityHandoffTurnKey{}, &capabilityHandoffTurnState{})
}

func capabilityHandoffTurnStateFromContext(ctx context.Context) *capabilityHandoffTurnState {
	if ctx == nil {
		return nil
	}
	st, _ := ctx.Value(capabilityHandoffTurnKey{}).(*capabilityHandoffTurnState)
	return st
}
