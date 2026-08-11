package maps

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// FreshLocationTTL is how long a session-shared snapshot can be reused
	// by maps_locate without a new desktop consent prompt.
	FreshLocationTTL = 2 * time.Minute
	// LocateRequestTTL is how long the desktop has to fulfill a pending locate.
	LocateRequestTTL = 30 * time.Second
)

// DeviceSnapshot is an ephemeral GPS reading. Never persist precise coords.
type DeviceSnapshot struct {
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	AccuracyM   float64   `json:"accuracy_m,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	CapturedAt  time.Time `json:"captured_at"`
	SessionID   string    `json:"session_id,omitempty"`
	Shared      bool      `json:"shared"`
	Source      string    `json:"source,omitempty"`
}

// SnapshotView is the API/tool payload, including computed age.
type SnapshotView struct {
	DeviceSnapshot
	AgeSeconds float64 `json:"age_s"`
}

// LocateRequest is a pending maps_locate consent prompt on the desktop.
type LocateRequest struct {
	ID        string     `json:"id"`
	AgentID   string     `json:"agent_id,omitempty"`
	AgentName string     `json:"agent_name,omitempty"`
	Channel   string     `json:"channel,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Status    string     `json:"status"`
	Reason    string     `json:"reason,omitempty"`
	Snapshot  *SnapshotView `json:"snapshot,omitempty"`
}

type locateWaiter struct {
	req  *LocateRequest
	done chan struct{}
}

// LocationStore holds the in-memory device location and pending locate requests.
type LocationStore struct {
	mu      sync.Mutex
	snap    *DeviceSnapshot
	pending map[string]*locateWaiter
}

// DefaultLocationStore is the process-wide ephemeral location cache.
var DefaultLocationStore = NewLocationStore()

// NewLocationStore returns an empty in-memory location store.
func NewLocationStore() *LocationStore {
	return &LocationStore{pending: make(map[string]*locateWaiter)}
}

// Publish stores a desktop-supplied snapshot. Shared marks session-share.
func (s *LocationStore) Publish(snap DeviceSnapshot) SnapshotView {
	if snap.CapturedAt.IsZero() {
		snap.CapturedAt = time.Now().UTC()
	}
	if snap.Source == "" {
		if snap.Shared {
			snap.Source = "session"
		} else {
			snap.Source = "locate"
		}
	}
	copySnap := snap
	s.mu.Lock()
	s.snap = &copySnap
	s.mu.Unlock()
	return viewOf(&copySnap)
}

// Clear drops the cached snapshot (chip off / session end).
func (s *LocationStore) Clear() {
	s.mu.Lock()
	s.snap = nil
	s.mu.Unlock()
}

// Get returns the last snapshot if any.
func (s *LocationStore) Get() (SnapshotView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snap == nil {
		return SnapshotView{}, false
	}
	return viewOf(s.snap), true
}

// FreshShared reports a session-shared snapshot younger than FreshLocationTTL.
func (s *LocationStore) FreshShared() (SnapshotView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snap == nil || !s.snap.Shared {
		return SnapshotView{}, false
	}
	view := viewOf(s.snap)
	if view.AgeSeconds > FreshLocationTTL.Seconds() {
		return SnapshotView{}, false
	}
	return view, true
}

// RequestLocate opens a pending desktop consent request.
func (s *LocationStore) RequestLocate(agentID, agentName, channel string) *LocateRequest {
	req := &LocateRequest{
		ID:        uuid.New().String()[:8],
		AgentID:   agentID,
		AgentName: agentName,
		Channel:   channel,
		CreatedAt: time.Now().UTC(),
		Status:    "pending",
	}
	s.mu.Lock()
	s.pending[req.ID] = &locateWaiter{req: req, done: make(chan struct{})}
	s.mu.Unlock()
	return req
}

// WaitLocate blocks until fulfill/reject or timeout.
func (s *LocationStore) WaitLocate(id string, timeout time.Duration) (*LocateRequest, error) {
	s.mu.Lock()
	w, ok := s.pending[id]
	s.mu.Unlock()
	if !ok || w == nil {
		return nil, fmt.Errorf("location request %s not found", id)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-w.done:
		s.mu.Lock()
		req := *w.req
		delete(s.pending, id)
		s.mu.Unlock()
		return &req, nil
	case <-timer.C:
		s.mu.Lock()
		if w.req.Status == "pending" {
			w.req.Status = "expired"
			w.req.Reason = "timed out waiting for location"
			closeDone(w.done)
		}
		req := *w.req
		delete(s.pending, id)
		s.mu.Unlock()
		return &req, fmt.Errorf("location request timed out")
	}
}

// Fulfill completes a pending locate with a new snapshot and caches it.
func (s *LocationStore) Fulfill(id string, snap DeviceSnapshot) (*LocateRequest, error) {
	if snap.CapturedAt.IsZero() {
		snap.CapturedAt = time.Now().UTC()
	}
	if snap.Source == "" {
		snap.Source = "locate"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.pending[id]
	if !ok || w == nil {
		return nil, fmt.Errorf("location request %s not found", id)
	}
	if w.req.Status != "pending" {
		return nil, fmt.Errorf("location request %s is %s", id, w.req.Status)
	}
	copySnap := snap
	if s.snap != nil && s.snap.Shared {
		copySnap.Shared = true
		copySnap.SessionID = s.snap.SessionID
	}
	s.snap = &copySnap
	view := viewOf(&copySnap)
	w.req.Status = "fulfilled"
	w.req.Snapshot = &view
	closeDone(w.done)
	out := *w.req
	return &out, nil
}

// Reject denies a pending locate.
func (s *LocationStore) Reject(id, reason string) (*LocateRequest, error) {
	if reason == "" {
		reason = "User declined to share location"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.pending[id]
	if !ok || w == nil {
		return nil, fmt.Errorf("location request %s not found", id)
	}
	if w.req.Status != "pending" {
		return nil, fmt.Errorf("location request %s is %s", id, w.req.Status)
	}
	w.req.Status = "rejected"
	w.req.Reason = reason
	closeDone(w.done)
	out := *w.req
	return &out, nil
}

// ListPending returns open locate requests for the desktop modal.
func (s *LocationStore) ListPending() []LocateRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]LocateRequest, 0, len(s.pending))
	for _, w := range s.pending {
		if w != nil && w.req != nil && w.req.Status == "pending" {
			out = append(out, *w.req)
		}
	}
	return out
}

func viewOf(snap *DeviceSnapshot) SnapshotView {
	age := time.Since(snap.CapturedAt).Seconds()
	if age < 0 {
		age = 0
	}
	return SnapshotView{DeviceSnapshot: *snap, AgeSeconds: age}
}

func closeDone(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}
