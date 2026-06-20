package agent

import (
	"sync"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CompressSnapshot accumulates compression stats for the current turn.
type CompressSnapshot struct {
	BytesIn  int
	BytesOut int
	Strategy string
	Refs     []string
}

type compressSnapshotHolder struct {
	mu   sync.Mutex
	snap CompressSnapshot
}

// resetCompressSnapshot clears compression stats for a new turn.
func (a *Agent) resetCompressSnapshot() {
	if a == nil {
		return
	}
	a.compressSnap.mu.Lock()
	a.compressSnap.snap = CompressSnapshot{}
	a.compressSnap.mu.Unlock()
}

// RecordCompressResult records one compression event.
func (a *Agent) RecordCompressResult(bytesIn, bytesOut int, strategy, ref string) {
	if a == nil {
		return
	}
	a.compressSnap.mu.Lock()
	defer a.compressSnap.mu.Unlock()
	a.compressSnap.snap.BytesIn += bytesIn
	a.compressSnap.snap.BytesOut += bytesOut
	if strategy != "" && strategy != "none" {
		a.compressSnap.snap.Strategy = strategy
	}
	if ref != "" {
		a.compressSnap.snap.Refs = append(a.compressSnap.snap.Refs, ref)
	}
}

// ApplyCompressMetadataToResponse stamps context_compress_* keys on the response.
func (a *Agent) ApplyCompressMetadataToResponse(msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	a.compressSnap.mu.Lock()
	snap := a.compressSnap.snap
	a.compressSnap.mu.Unlock()
	if snap.BytesIn == 0 && snap.BytesOut == 0 {
		return
	}
	refs := ""
	for i, r := range snap.Refs {
		if i > 0 {
			refs += ","
		}
		refs += r
	}
	protocol.ApplyCompressMeta(msg, protocol.CompressMeta{
		BytesIn:  snap.BytesIn,
		BytesOut: snap.BytesOut,
		Strategy: snap.Strategy,
		Refs:     refs,
	})
}
