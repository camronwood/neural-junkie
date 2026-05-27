package collaboration

// cloneWorkspaceContextMap returns a shallow copy safe to store on a collaboration.
func cloneWorkspaceContextMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
