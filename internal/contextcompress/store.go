package contextcompress

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store caches original content for reversible compression (CCR).
type Store struct {
	mu         sync.Mutex
	entries    map[string]*cacheEntry
	order      []string
	maxEntries int
	ttl        time.Duration
	diskDir    string
}

type cacheEntry struct {
	data      []byte
	channelID string
	createdAt time.Time
}

// NewStore creates an LRU cache with optional disk spill directory.
func NewStore(maxEntries int, ttlMinutes int, diskDir string) *Store {
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}
	if ttlMinutes <= 0 {
		ttlMinutes = defaultTTLMinutes
	}
	return &Store{
		entries:    make(map[string]*cacheEntry),
		maxEntries: maxEntries,
		ttl:        time.Duration(ttlMinutes) * time.Minute,
		diskDir:    diskDir,
	}
}

var (
	defaultStore   *Store
	defaultStoreMu sync.RWMutex
)

// SetDefaultStore wires the hub-owned cache (call at hub startup).
func SetDefaultStore(s *Store) {
	defaultStoreMu.Lock()
	defaultStore = s
	defaultStoreMu.Unlock()
}

// DefaultStore returns the hub cache or an ephemeral in-process store.
func DefaultStore() *Store {
	defaultStoreMu.RLock()
	s := defaultStore
	defaultStoreMu.RUnlock()
	if s != nil {
		return s
	}
	return NewStore(defaultMaxEntries, defaultTTLMinutes, "")
}

// Put stores original bytes and returns a stable ref id.
func (s *Store) Put(channelID, callID, label string, original []byte) string {
	if s == nil || len(original) == 0 {
		return ""
	}
	ref := makeRef(channelID, callID, label, original)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictExpiredLocked()
	if _, ok := s.entries[ref]; !ok {
		s.order = append(s.order, ref)
	}
	s.entries[ref] = &cacheEntry{
		data:      append([]byte(nil), original...),
		channelID: channelID,
		createdAt: time.Now(),
	}
	s.enforceCapacityLocked()
	s.writeDiskLocked(ref, original)
	return ref
}

// Get returns cached original content.
func (s *Store) Get(ref string) ([]byte, bool) {
	if s == nil || ref == "" {
		return nil, false
	}
	s.mu.Lock()
	e, ok := s.entries[ref]
	if ok && s.ttl > 0 && time.Since(e.createdAt) > s.ttl {
		delete(s.entries, ref)
		s.removeOrderLocked(ref)
		ok = false
		e = nil
	}
	var data []byte
	if ok && e != nil {
		data = append([]byte(nil), e.data...)
	}
	s.mu.Unlock()
	if ok {
		return data, true
	}
	if disk := s.readDisk(ref); len(disk) > 0 {
		return disk, true
	}
	return nil, false
}

func makeRef(channelID, callID, label string, original []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(channelID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(callID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(label))
	_, _ = h.Write(original)
	sum := hex.EncodeToString(h.Sum(nil))
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return "ctx-" + sum
}

func (s *Store) enforceCapacityLocked() {
	for len(s.order) > s.maxEntries {
		old := s.order[0]
		s.order = s.order[1:]
		delete(s.entries, old)
	}
}

func (s *Store) evictExpiredLocked() {
	if s.ttl <= 0 {
		return
	}
	now := time.Now()
	kept := s.order[:0]
	for _, ref := range s.order {
		e, ok := s.entries[ref]
		if !ok || now.Sub(e.createdAt) > s.ttl {
			delete(s.entries, ref)
			continue
		}
		kept = append(kept, ref)
	}
	s.order = kept
}

func (s *Store) removeOrderLocked(ref string) {
	for i, r := range s.order {
		if r == ref {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}

func (s *Store) writeDiskLocked(ref string, data []byte) {
	if s.diskDir == "" {
		return
	}
	path := filepath.Join(s.diskDir, ref+".bin")
	_ = os.MkdirAll(s.diskDir, 0o700)
	_ = os.WriteFile(path, data, 0o600)
}

func (s *Store) readDisk(ref string) []byte {
	if s.diskDir == "" {
		return nil
	}
	path := filepath.Join(s.diskDir, ref+".bin")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

// DefaultCacheDir returns ~/.neural-junkie/context-cache.
func DefaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".neural-junkie", "context-cache")
}
