package agent

import "context"

// hubArenaNoop provides default Model Arena HubClient methods for test stubs.
type hubArenaNoop struct{}

func (hubArenaNoop) ArenaEnabled() bool { return false }

func (hubArenaNoop) ArenaSidecarGet(context.Context, string) (map[string]any, error) {
	return nil, nil
}

func (hubArenaNoop) ArenaSidecarPost(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
