package agent

import "context"

type askUserTurnKey struct{}

type askUserTurnState struct {
	count    int
	answered []string
}

func withAskUserTurnState(ctx context.Context) context.Context {
	if askUserTurnStateFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, askUserTurnKey{}, &askUserTurnState{})
}

// withImplementationSessionTurnState creates the ask_user budget once at the
// implementation-session boundary. Child tool rounds inherit this context.
func withImplementationSessionTurnState(ctx context.Context) context.Context {
	return withAskUserTurnState(ctx)
}

func askUserTurnStateFromContext(ctx context.Context) *askUserTurnState {
	if ctx == nil {
		return nil
	}
	st, _ := ctx.Value(askUserTurnKey{}).(*askUserTurnState)
	return st
}

// pinGenerationForUserWait prevents AbortChannel from cancelling this agent while
// blocked inside ask_user (peer pause must not kill the waiter).
func (a *Agent) pinGenerationForUserWait(channel string) {
	if a == nil || channel == "" {
		return
	}
	a.userWaitMu.Lock()
	defer a.userWaitMu.Unlock()
	if a.userWaitPins == nil {
		a.userWaitPins = make(map[string]int)
	}
	a.userWaitPins[channel]++
}

func (a *Agent) unpinGenerationForUserWait(channel string) {
	if a == nil || channel == "" {
		return
	}
	a.userWaitMu.Lock()
	defer a.userWaitMu.Unlock()
	if a.userWaitPins == nil {
		return
	}
	if a.userWaitPins[channel] <= 1 {
		delete(a.userWaitPins, channel)
		return
	}
	a.userWaitPins[channel]--
}

func (a *Agent) isGenerationPinnedForUserWait(channel string) bool {
	if a == nil || channel == "" {
		return false
	}
	a.userWaitMu.Lock()
	defer a.userWaitMu.Unlock()
	return a.userWaitPins[channel] > 0
}
