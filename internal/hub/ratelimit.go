package hub

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// RateLimiter is a simple sliding-window limiter per client key.
type RateLimiter struct {
	mu      sync.Mutex
	events  map[string][]time.Time
	enabled bool
	readMax int
	mutMax  int
	window  time.Duration
}

func NewRateLimiter() *RateLimiter {
	sec := config.AppConfig().ResolvedSecurity()
	rl := &RateLimiter{
		events:  make(map[string][]time.Time),
		enabled: sec.RateLimitEnabledOrDefault(),
		readMax: sec.RateReadPerMinute,
		mutMax:  sec.RateMutatePerMinute,
		window:  time.Minute,
	}
	if rl.readMax <= 0 {
		rl.readMax = 300
	}
	if rl.mutMax <= 0 {
		rl.mutMax = 120
	}
	if os.Getenv("NEURAL_JUNKIE_RATE_LIMIT") == "0" {
		rl.enabled = false
	}
	return rl
}

// Reconfigure updates rate limit settings at runtime.
func (rl *RateLimiter) Reconfigure(enabled bool, readMax, mutMax int) {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.enabled = enabled
	if readMax > 0 {
		rl.readMax = readMax
	}
	if mutMax > 0 {
		rl.mutMax = mutMax
	}
}

func (rl *RateLimiter) clientKey(r *http.Request) string {
	if tok := ExtractSessionToken(r); tok != "" {
		return "sess:" + tok
	}
	if tok := ExtractHubToken(r); tok != "" {
		return "hub:" + tok
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimitExempt reports endpoints that should not consume the global rate budget.
func rateLimitExempt(r *http.Request) bool {
	path := ""
	if r.URL != nil {
		path = r.URL.Path
	}
	// Sidebar agent clicks use find-or-create DM; must not starve /api/send.
	if r.Method == http.MethodPost && path == "/api/channels/open-dm" {
		return true
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	switch path {
	case "/api/ollama/catalog", "/api/hf/catalog", "/api/ollama/install-status":
		return true
	}
	if strings.HasPrefix(path, "/api/ollama/library/") {
		return true
	}
	return false
}

func (rl *RateLimiter) Allow(r *http.Request) bool {
	if rl == nil || !rl.enabled {
		return true
	}
	if rateLimitExempt(r) {
		return true
	}
	limit := rl.readMax
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		limit = rl.mutMax
	}
	// Sensitive endpoints may override the global budgets.
	if r.URL != nil && strings.HasPrefix(r.URL.Path, "/api/room/join") {
		// Guests do not have a session yet, so the key is often RemoteAddr.
		// Keep this conservative to discourage brute-forcing 6-char join codes.
		limit = 20
	}
	key := rl.clientKey(r)
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()
	q := rl.events[key]
	j := 0
	for _, t := range q {
		if t.After(cutoff) {
			q[j] = t
			j++
		}
	}
	q = q[:j]
	if len(q) >= limit {
		return false
	}
	rl.events[key] = append(q, now)
	return true
}

// RateLimitMiddleware returns 429 when over limit.
func RateLimitMiddleware(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
