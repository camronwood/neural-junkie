package hub

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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
	enabled := os.Getenv("NEURAL_JUNKIE_RATE_LIMIT") != "0"
	readMax := 300
	mutMax := 120
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_RATE_READ")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			readMax = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_RATE_MUTATE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			mutMax = n
		}
	}
	return &RateLimiter{
		events:  make(map[string][]time.Time),
		enabled: enabled,
		readMax: readMax,
		mutMax:  mutMax,
		window:  time.Minute,
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

// rateLimitExempt reports read-only catalog endpoints that should not consume the global GET budget.
func rateLimitExempt(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	switch r.URL.Path {
	case "/api/ollama/catalog", "/api/hf/catalog", "/api/ollama/install-status":
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/ollama/library/") {
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
