package capabilities

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	globalMu       sync.RWMutex
	globalProfiles *Profiles
)

// Global returns the loaded capability profiles (may be nil).
func Global() *Profiles {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalProfiles
}

// SetGlobal replaces the runtime capability profiles (tests).
func SetGlobal(p *Profiles) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalProfiles = p
}

// Load initializes capability profiles from env, repo path, or embedded default.
func Load() (*Profiles, error) {
	candidates := []string{}
	if path := stringsTrim(os.Getenv("NEURAL_JUNKIE_CAPABILITY_PROFILES")); path != "" {
		candidates = append(candidates, path)
	}
	if root := findRepoRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "docs", "data", "model-capability-profiles.json"))
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		p, err := ParseProfiles(data)
		if err != nil {
			log.Printf("[capabilities] skip %s: %v", path, err)
			continue
		}
		globalMu.Lock()
		globalProfiles = p
		globalMu.Unlock()
		log.Printf("[capabilities] loaded from %s (source_run=%s)", path, p.SourceRunID)
		return p, nil
	}

	p, err := ParseProfiles(embeddedJSON)
	if err != nil {
		return nil, err
	}
	globalMu.Lock()
	globalProfiles = p
	globalMu.Unlock()
	log.Printf("[capabilities] using embedded profiles (source_run=%s)", p.SourceRunID)
	return p, nil
}

func findRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "docs", "data")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
