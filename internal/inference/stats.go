package inference

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
)

const maxRecentTurns = 200

// TurnRecord is one inference turn persisted for dashboard rollups.
type TurnRecord struct {
	At               time.Time `json:"at"`
	Channel          string    `json:"channel"`
	AgentID          string    `json:"agent_id"`
	AgentName        string    `json:"agent_name"`
	ProviderID       string    `json:"provider_id,omitempty"`
	Model            string    `json:"model,omitempty"`
	CostTier         string    `json:"cost_tier,omitempty"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	Calls            int       `json:"calls,omitempty"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd,omitempty"`
	TTFTMs           float64   `json:"ttft_ms,omitempty"`
	TokPerS          float64   `json:"tok_per_s,omitempty"`
}

// BucketTotals aggregates token and cost stats for a grouping key.
type BucketTotals struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Calls            int     `json:"calls"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

// Summary is the dashboard API payload.
type Summary struct {
	StartedAt  time.Time              `json:"started_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Totals     BucketTotals             `json:"totals"`
	ByProvider map[string]BucketTotals  `json:"by_provider"`
	ByModel    map[string]BucketTotals  `json:"by_model"`
	Recent     []TurnRecord             `json:"recent"`
}

// StatsStore accumulates inference usage with optional JSON persistence.
type StatsStore struct {
	mu       sync.RWMutex
	path     string
	started  time.Time
	updated  time.Time
	totals   BucketTotals
	byProv   map[string]BucketTotals
	byModel  map[string]BucketTotals
	recent   []TurnRecord
}

var defaultStore *StatsStore

// SetDefaultStore installs the process-wide usage store (called from cmd/server).
func SetDefaultStore(s *StatsStore) {
	defaultStore = s
}

// DefaultStore returns the process-wide store (may be nil before server init).
func DefaultStore() *StatsStore {
	return defaultStore
}

// NewStatsStore loads or creates a store at path (empty path = in-memory only).
func NewStatsStore(path string) (*StatsStore, error) {
	s := &StatsStore{
		path:    path,
		started: time.Now().UTC(),
		updated: time.Now().UTC(),
		byProv:  make(map[string]BucketTotals),
		byModel: make(map[string]BucketTotals),
	}
	if path != "" {
		if err := s.load(); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return s, nil
}

func (s *StatsStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var snap Summary
	if err := json.Unmarshal(raw, &snap); err != nil {
		return err
	}
	s.started = snap.StartedAt
	s.updated = snap.UpdatedAt
	s.totals = snap.Totals
	if snap.ByProvider != nil {
		s.byProv = snap.ByProvider
	}
	if snap.ByModel != nil {
		s.byModel = snap.ByModel
	}
	s.recent = snap.Recent
	return nil
}

func (s *StatsStore) persistLocked() error {
	if s.path == "" {
		return nil
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	snap := s.summaryLocked()
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func addBucket(b *BucketTotals, rec TurnRecord) {
	b.PromptTokens += rec.PromptTokens
	b.CompletionTokens += rec.CompletionTokens
	if rec.Calls > 0 {
		b.Calls += rec.Calls
	} else if rec.PromptTokens > 0 || rec.CompletionTokens > 0 {
		b.Calls++
	}
	b.EstimatedCostUSD += rec.EstimatedCostUSD
}

// Record appends a turn and updates rollups.
func (s *StatsStore) Record(rec TurnRecord) {
	if s == nil {
		return
	}
	if rec.At.IsZero() {
		rec.At = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updated = rec.At
	addBucket(&s.totals, rec)
	if rec.ProviderID != "" {
		b := s.byProv[rec.ProviderID]
		addBucket(&b, rec)
		s.byProv[rec.ProviderID] = b
	}
	if rec.Model != "" {
		b := s.byModel[rec.Model]
		addBucket(&b, rec)
		s.byModel[rec.Model] = b
	}
	s.recent = append(s.recent, rec)
	if len(s.recent) > maxRecentTurns {
		s.recent = s.recent[len(s.recent)-maxRecentTurns:]
	}
	_ = s.persistLocked()
}

// Reset clears all stored stats.
func (s *StatsStore) Reset() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.started = now
	s.updated = now
	s.totals = BucketTotals{}
	s.byProv = make(map[string]BucketTotals)
	s.byModel = make(map[string]BucketTotals)
	s.recent = nil
	if s.path != "" {
		return os.Remove(s.path)
	}
	return nil
}

func (s *StatsStore) summaryLocked() Summary {
	return Summary{
		StartedAt:  s.started,
		UpdatedAt:  s.updated,
		Totals:     s.totals,
		ByProvider: copyBuckets(s.byProv),
		ByModel:    copyBuckets(s.byModel),
		Recent:     append([]TurnRecord(nil), s.recent...),
	}
}

func copyBuckets(in map[string]BucketTotals) map[string]BucketTotals {
	if len(in) == 0 {
		return map[string]BucketTotals{}
	}
	out := make(map[string]BucketTotals, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Summary returns a snapshot for API responses.
func (s *StatsStore) Summary() Summary {
	if s == nil {
		return Summary{
			ByProvider: map[string]BucketTotals{},
			ByModel:    map[string]BucketTotals{},
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.summaryLocked()
}

// TurnRecordFromUsage builds a TurnRecord from pipeline context.
func TurnRecordFromUsage(
	channel, agentID, agentName, providerID, model, costTier string,
	u ai.InferenceUsage,
	costUSD float64,
) TurnRecord {
	calls := u.Calls
	if calls == 0 && (u.PromptTokens > 0 || u.CompletionTokens > 0) {
		calls = 1
	}
	rec := TurnRecord{
		At:               time.Now().UTC(),
		Channel:          channel,
		AgentID:          agentID,
		AgentName:        agentName,
		ProviderID:       providerID,
		Model:            model,
		CostTier:         costTier,
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		Calls:            calls,
		EstimatedCostUSD: costUSD,
	}
	if ttft := u.TTFTMs(); ttft > 0 {
		rec.TTFTMs = ttft
	}
	if tps := u.TokPerS(); tps > 0 {
		rec.TokPerS = tps
	}
	return rec
}
