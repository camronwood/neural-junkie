package hub

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

const bootstrapTokenFileName = "bootstrap.token"

// BootstrapTokenPath returns ~/.neural-junkie/bootstrap.token unless overridden.
func BootstrapTokenPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN_FILE")); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", bootstrapTokenFileName), nil
}

// EnsureBootstrapToken creates bootstrap.token on first hub start when missing.
func EnsureBootstrapToken() {
	if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN")) != "" {
		return
	}
	path, err := BootstrapTokenPath()
	if err != nil {
		log.Printf("[auth] bootstrap token path: %v", err)
		return
	}
	if _, err := os.Stat(path); err == nil {
		return
	} else if !os.IsNotExist(err) {
		log.Printf("[auth] bootstrap token stat: %v", err)
		return
	}
	token, err := newBootstrapSecret()
	if err != nil {
		log.Printf("[auth] bootstrap token generate: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		log.Printf("[auth] bootstrap token mkdir: %v", err)
		return
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		log.Printf("[auth] bootstrap token write: %v", err)
		return
	}
	log.Printf("[auth] created hub bootstrap token at %s (use X-NJ-Bootstrap or NEURAL_JUNKIE_BOOTSTRAP_TOKEN to mint admin sessions)", path)
}

func newBootstrapSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func configuredBootstrapToken() string {
	return strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN"))
}

func loadBootstrapTokenFromDisk() string {
	path, err := BootstrapTokenPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ExtractBootstrapToken reads X-NJ-Bootstrap or Authorization: Bearer (non-nj_, non-hub).
func ExtractBootstrapToken(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get("X-NJ-Bootstrap")); h != "" {
		return h
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return ""
	}
	tok := strings.TrimSpace(auth[7:])
	if tok == "" || strings.HasPrefix(tok, "nj_") {
		return ""
	}
	if HubTokenConfigured() && subtle.ConstantTimeCompare([]byte(tok), []byte(HubAccessToken())) == 1 {
		return ""
	}
	return tok
}

// ValidBootstrapToken reports whether the request presents the hub bootstrap secret.
func ValidBootstrapToken(r *http.Request) bool {
	got := ExtractBootstrapToken(r)
	if got == "" {
		return false
	}
	want := configuredBootstrapToken()
	if want == "" {
		want = loadBootstrapTokenFromDisk()
	}
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// RelaxedLocal enables loopback-only synthetic member sessions (dev/Makefile escape hatch).
func RelaxedLocal() bool {
	if config.AppConfig().ResolvedSecurity().RelaxedLocal {
		return true
	}
	return strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_RELAXED_LOCAL")) == "1"
}

// BootstrapConfigured reports whether an admin bootstrap secret is available.
func BootstrapConfigured() bool {
	if configuredBootstrapToken() != "" {
		return true
	}
	return loadBootstrapTokenFromDisk() != ""
}
