// Package relay implements the public HTTPS Slack OAuth redirect relay for Neural Junkie.
package relay

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
)

// Handler serves Slack OAuth callback paths and forwards the browser to the local hub.
type Handler struct {
	// AllowedPaths limits which relay paths are served (defaults to bot + user-dm callbacks).
	AllowedPaths map[string]bool
}

// NewHandler returns a relay handler with default callback paths.
func NewHandler() *Handler {
	return &Handler{
		AllowedPaths: map[string]bool{
			slackint.OAuthCallbackPath:       true,
			slackint.UserDMOAuthCallbackPath: true,
		},
	}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if h.AllowedPaths != nil && !h.AllowedPaths[path] {
		http.NotFound(w, r)
		return
	}
	if err := h.forward(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *Handler) forward(w http.ResponseWriter, r *http.Request) error {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		return fmt.Errorf("missing state")
	}
	_, localReturn, ok := slackint.ParseOAuthState(state)
	if !ok || localReturn == "" {
		return fmt.Errorf("invalid state — start Connect Slack from Neural Junkie again")
	}
	target, err := slackint.BuildRelayRedirectURL(localReturn, r.URL.Query())
	if err != nil {
		return err
	}
	http.Redirect(w, r, target, http.StatusFound)
	return nil
}

// HealthResponse is returned by GET /healthz.
type HealthResponse struct {
	OK      bool   `json:"ok"`
	Service string `json:"service"`
}

// HandleHealthz writes a simple health check response.
func HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true,"service":"nj-slack-oauth-relay"}`))
}

// Mux returns an http.ServeMux wired for the relay service.
func Mux() *http.ServeMux {
	h := NewHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HandleHealthz)
	mux.Handle(slackint.OAuthCallbackPath, h)
	mux.Handle(slackint.UserDMOAuthCallbackPath, h)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("Neural Junkie Slack OAuth relay\n"))
			return
		}
		// Normalize path for nested API Gateway stages if present.
		path := r.URL.Path
		if strings.HasSuffix(path, slackint.OAuthCallbackPath) {
			r2 := r.Clone(r.Context())
			u, _ := url.Parse(r.URL.String())
			u.Path = slackint.OAuthCallbackPath
			r2.URL = u
			h.ServeHTTP(w, r2)
			return
		}
		if strings.HasSuffix(path, slackint.UserDMOAuthCallbackPath) {
			r2 := r.Clone(r.Context())
			u, _ := url.Parse(r.URL.String())
			u.Path = slackint.UserDMOAuthCallbackPath
			r2.URL = u
			h.ServeHTTP(w, r2)
			return
		}
		http.NotFound(w, r)
	})
	return mux
}
