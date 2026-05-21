package hub

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// HubSession is a desktop/user session for ACL and auditing.
type HubSession struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionManager issues and validates hub user sessions (in-memory; restart clears).
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*HubSession // token -> session
	ttl      time.Duration
}

func NewSessionManager() *SessionManager {
	ttl := 7 * 24 * time.Hour
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SESSION_TTL_HOURS")); v != "" {
		if h, err := time.ParseDuration(v + "h"); err == nil && h > 0 {
			ttl = h
		}
	}
	return &SessionManager{
		sessions: make(map[string]*HubSession),
		ttl:      ttl,
	}
}

// AuthRequired returns true when mutations must carry a valid session (strict mode).
func AuthRequired() bool {
	return strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_AUTH_REQUIRED")) == "1"
}

// EnforceChannelACL returns true when channel access should be checked for the request.
func EnforceChannelACL(r *http.Request) bool {
	if AuthRequired() {
		return true
	}
	return ExtractSessionToken(r) != ""
}

func slugUsername(username string) string {
	s := strings.ToLower(strings.TrimSpace(username))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "anonymous"
	}
	return out
}

// CreateSession registers a user session and returns the token.
func (sm *SessionManager) CreateSession(username string) *HubSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.pruneLocked()
	token := newSessionToken()
	user := slugUsername(username)
	s := &HubSession{
		Token:     token,
		UserID:    user,
		Username:  strings.TrimSpace(username),
		ExpiresAt: time.Now().Add(sm.ttl),
	}
	if s.Username == "" {
		s.Username = "Anonymous"
	}
	sm.sessions[token] = s
	return s
}

func newSessionToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (sm *SessionManager) pruneLocked() {
	now := time.Now()
	for tok, s := range sm.sessions {
		if s == nil || now.After(s.ExpiresAt) {
			delete(sm.sessions, tok)
		}
	}
}

// Validate returns the session for a token or nil.
func (sm *SessionManager) Validate(token string) *HubSession {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[token]
	if !ok || s == nil || time.Now().After(s.ExpiresAt) {
		return nil
	}
	return s
}

// ExtractSessionToken reads X-NJ-Session or Authorization: Bearer (after hub token check).
func ExtractSessionToken(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get("X-NJ-Session")); h != "" {
		return h
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	// If both hub token and session use Bearer, prefer X-NJ-Session; hub token uses X-NJ-Hub-Token
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		tok := strings.TrimSpace(auth[7:])
		if HubTokenConfigured() && tok == HubAccessToken() {
			return ""
		}
		// Session may also be sent as Bearer when hub token is not configured
		if !HubTokenConfigured() {
			return tok
		}
	}
	return ""
}

// SessionFromRequest validates session header against the manager.
func SessionFromRequest(r *http.Request, sm *SessionManager) *HubSession {
	if sm == nil {
		return nil
	}
	return sm.Validate(ExtractSessionToken(r))
}

// RequireSessionForMutation ensures a valid session when auth is required or a session header was sent.
func RequireSessionForMutation(w http.ResponseWriter, r *http.Request, sm *SessionManager) (*HubSession, bool) {
	sess := SessionFromRequest(r, sm)
	if sess != nil {
		return sess, true
	}
	if AuthRequired() || ExtractSessionToken(r) != "" {
		http.Error(w, "Unauthorized: valid X-NJ-Session required (POST /api/auth/session)", http.StatusUnauthorized)
		return nil, false
	}
	// Relaxed local mode: loopback clients may omit session
	if AllowHubRequest(r) {
		return &HubSession{UserID: "local", Username: "local"}, true
	}
	http.Error(w, "Forbidden", http.StatusForbidden)
	return nil, false
}
