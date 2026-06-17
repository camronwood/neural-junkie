package workspacebackend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HealthCheck pings sidecar GET /health. Local backends always return nil.
func HealthCheck(ctx context.Context, b Backend) error {
	if b == nil {
		return nil
	}
	rb, ok := b.(*RemoteBackend)
	if !ok {
		return nil
	}
	url := strings.TrimRight(rb.sidecarURL, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if rb.token != "" {
		req.Header.Set("Authorization", "Bearer "+rb.token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sidecar health %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}
