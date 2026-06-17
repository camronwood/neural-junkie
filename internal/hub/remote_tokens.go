package hub

import "github.com/camronwood/neural-junkie/internal/remotetokens"

// SaveRemoteToken stores sidecar bearer token for a workspace ID.
func SaveRemoteToken(workspaceID, token string) error {
	return remotetokens.Save(workspaceID, token)
}

// GetRemoteToken returns stored sidecar token.
func GetRemoteToken(workspaceID string) (string, error) {
	return remotetokens.Get(workspaceID)
}

// DeleteRemoteToken removes token for workspace.
func DeleteRemoteToken(workspaceID string) error {
	return remotetokens.Delete(workspaceID)
}

// ListRemoteTokens returns workspace ID → token map copy.
func ListRemoteTokens() (map[string]string, error) {
	return remotetokens.List()
}

// RemoteTokenStorePath returns path for diagnostics.
func RemoteTokenStorePath() (string, error) {
	return remotetokens.Path()
}
