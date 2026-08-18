package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	prepareTokenMetaKey   = "prepare_token"
	prepareTokenTTL       = 2 * time.Minute
	maxPreparedTurns      = 256
)

type preparedTurn struct {
	Decision      intent.TurnDecision
	Channel       string
	Content       string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	ContextRequest intent.ContextRequest
}

// PrepareTurnResult is returned by /api/turn/prepare.
type PrepareTurnResult struct {
	PrepareToken   string                 `json:"prepare_token"`
	ContextRequest intent.ContextRequest  `json:"context_request"`
	Decision       intent.TurnDecision    `json:"decision"`
}

func newPrepareToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (h *Hub) ensurePreparedTurns() {
	if h.preparedTurns == nil {
		h.preparedTurns = make(map[string]*preparedTurn)
	}
}

// PrepareTurn classifies a provisional turn from a structural availability
// envelope and returns a context_request for the client to upload payloads.
func (h *Hub) PrepareTurn(ctx context.Context, msg *protocol.Message) (*PrepareTurnResult, error) {
	if h == nil || msg == nil {
		return nil, errPrepareUnavailable
	}
	if msg.Metadata != nil {
		delete(msg.Metadata, protocol.MetadataTurnDecision)
		delete(msg.Metadata, protocol.TurnMetaGovernance)
		delete(msg.Metadata, prepareTokenMetaKey)
	}
	if !semanticTurnEligible(msg) {
		return nil, errPrepareIneligible
	}
	h.mu.RLock()
	router := h.semanticTurnRouter
	h.mu.RUnlock()
	if router == nil {
		return nil, errPrepareUnavailable
	}

	stampCanonicalGovernance(msg)
	features := h.semanticTurnFeatures(msg)
	decision := router.Resolve(ctx, features)
	intent.EnsureContextPlan(&decision, features)
	req := intent.ContextRequestFromPlan(decision.ContextPlan, features)

	token, err := newPrepareToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	h.mu.Lock()
	h.ensurePreparedTurns()
	h.prunePreparedTurnsLocked(now)
	if len(h.preparedTurns) >= maxPreparedTurns {
		h.mu.Unlock()
		return nil, errPrepareBusy
	}
	h.preparedTurns[token] = &preparedTurn{
		Decision:       decision,
		Channel:        msg.Channel,
		Content:        msg.Content,
		CreatedAt:      now,
		ExpiresAt:      now.Add(prepareTokenTTL),
		ContextRequest: req,
	}
	h.mu.Unlock()

	return &PrepareTurnResult{
		PrepareToken:   token,
		ContextRequest: req,
		Decision:       decision,
	}, nil
}

// PeekPreparedTurn returns a copy of the prepared decision without consuming it.
func (h *Hub) PeekPreparedTurn(token string) (*preparedTurn, bool) {
	token = trimPrepareToken(token)
	if h == nil || token == "" {
		return nil, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensurePreparedTurns()
	h.prunePreparedTurnsLocked(time.Now())
	pt, ok := h.preparedTurns[token]
	if !ok || pt == nil {
		return nil, false
	}
	cp := *pt
	return &cp, true
}

// ConsumePreparedTurn removes and returns a valid prepared turn.
func (h *Hub) ConsumePreparedTurn(token string) (*preparedTurn, bool) {
	token = trimPrepareToken(token)
	if h == nil || token == "" {
		return nil, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensurePreparedTurns()
	h.prunePreparedTurnsLocked(time.Now())
	pt, ok := h.preparedTurns[token]
	if !ok || pt == nil {
		return nil, false
	}
	delete(h.preparedTurns, token)
	return pt, true
}

func (h *Hub) prunePreparedTurnsLocked(now time.Time) {
	for token, pt := range h.preparedTurns {
		if pt == nil || now.After(pt.ExpiresAt) {
			delete(h.preparedTurns, token)
		}
	}
}

func trimPrepareToken(token string) string {
	for len(token) > 0 && (token[0] == ' ' || token[0] == '\t') {
		token = token[1:]
	}
	for len(token) > 0 && (token[len(token)-1] == ' ' || token[len(token)-1] == '\t') {
		token = token[:len(token)-1]
	}
	return token
}

func prepareTokenFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	raw, ok := msg.Metadata[prepareTokenMetaKey]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return trimPrepareToken(v)
	default:
		return ""
	}
}

var (
	errPrepareUnavailable = errString("semantic prepare unavailable")
	errPrepareIneligible  = errString("turn not eligible for semantic prepare")
	errPrepareBusy        = errString("too many pending prepared turns")
)

type errString string

func (e errString) Error() string { return string(e) }
