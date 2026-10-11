package hub

import (
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

// HubAccessToken returns the optional shared secret for non-loopback hub API access.
func HubAccessToken() string {
	sec := config.AppConfig().ResolvedSecurity()
	if t := strings.TrimSpace(sec.HubToken); t != "" {
		return t
	}
	return strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_HUB_TOKEN"))
}

// HubTokenConfigured reports whether remote clients may authenticate with a token.
func HubTokenConfigured() bool {
	return HubAccessToken() != ""
}

// allowHubTokenQuery reports whether ?hub_token= may authenticate a request.
// Prefer X-NJ-Hub-Token / Authorization headers. Query tokens are accepted only
// for WebSocket upgrades (browsers cannot set custom headers) or when
// NEURAL_JUNKIE_RELAXED_LOCAL=1 (dev escape hatch).
func allowHubTokenQuery(r *http.Request) bool {
	if RelaxedLocal() {
		return true
	}
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// ExtractHubToken reads X-NJ-Hub-Token or Authorization: Bearer.
// ?hub_token= is accepted only when allowHubTokenQuery permits it.
func ExtractHubToken(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get("X-NJ-Hub-Token")); h != "" {
		return h
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if q := strings.TrimSpace(r.URL.Query().Get("hub_token")); q != "" {
		if allowHubTokenQuery(r) {
			return q
		}
	}
	return ""
}

// IsLoopbackRemoteAddr reports whether the TCP peer is on this machine.
func IsLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// AllowHubRequest permits the request when it is from loopback or presents a valid hub token.
func AllowHubRequest(r *http.Request) bool {
	if tok := HubAccessToken(); tok != "" {
		got := ExtractHubToken(r)
		if got != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(got)) == 1 {
			return true
		}
	}
	return IsLoopbackRemoteAddr(r.RemoteAddr)
}

// RequireHubAccess writes 403 and returns false when the request is not allowed.
func RequireHubAccess(w http.ResponseWriter, r *http.Request) bool {
	if AllowHubRequest(r) {
		return true
	}
	http.Error(
		w,
		"Forbidden: hub API is only available from this machine unless NEURAL_JUNKIE_HUB_TOKEN is configured",
		http.StatusForbidden,
	)
	return false
}
