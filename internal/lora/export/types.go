package export

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SourceKind selects where training rows are collected from.
type SourceKind string

const (
	SourceChannel       SourceKind = "channel"
	SourceCollaboration SourceKind = "collaboration"
	SourceRepo          SourceKind = "repo"
)

const (
	DefaultMaxRows      = 2000
	MinRows             = 10
	DefaultRefreshDelta = 20
)

// Row is one Alpaca-style training example.
type Row struct {
	RowID       string `json:"row_id,omitempty"`
	Instruction string `json:"instruction"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	SourceKind  string `json:"source_kind,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
	MessageAt   string `json:"message_at,omitempty"`
	Included    bool   `json:"included,omitempty"`
}

// Request describes a dataset export.
type Request struct {
	Source            SourceKind
	SourceID          string
	ThreadID          string
	AgentName         string
	MaxRows           int
	ExtraRows         []Row
	SinceTimestamp    time.Time
	SinceJobExportedAt time.Time
	ApprovedTasksOnly bool
	RowIDs            []string
	PriorAdapterID    string
}

// ContentHash returns a stable hash for deduplication.
func ContentHash(instruction, input, output string) string {
	norm := strings.ToLower(strings.TrimSpace(instruction)) + "\n" +
		strings.ToLower(strings.TrimSpace(input)) + "\n" +
		strings.ToLower(strings.TrimSpace(output))
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}

func rowID(parts ...string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(strings.TrimSpace(p))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

func applyRowFilters(rows []PreviewRow, req Request) []PreviewRow {
	if len(req.RowIDs) == 0 {
		for i := range rows {
			rows[i].Included = true
		}
		return dedupeRows(rows)
	}
	allow := make(map[string]struct{}, len(req.RowIDs))
	for _, id := range req.RowIDs {
		allow[strings.TrimSpace(id)] = struct{}{}
	}
	out := make([]PreviewRow, 0, len(req.RowIDs))
	for _, r := range rows {
		if _, ok := allow[r.RowID]; ok {
			r.Included = true
			out = append(out, r)
		}
	}
	return dedupeRows(out)
}

func dedupeRows(rows []PreviewRow) []PreviewRow {
	seen := make(map[string]struct{})
	out := make([]PreviewRow, 0, len(rows))
	for _, r := range rows {
		h := ContentHash(r.Instruction, r.Input, r.Output)
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		if r.RowID == "" {
			r.RowID = rowID(r.SourceKind, r.SourceRef, r.Instruction, r.Output)
		}
		out = append(out, r)
	}
	return out
}

// PreviewRow is a row with metadata for curation UI.
type PreviewRow struct {
	Row
}

// RowsToAlpaca strips metadata for training.
func RowsToAlpaca(rows []PreviewRow) []Row {
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		if !r.Included && len(rows) > 0 && r.Included == false {
			// when Included unset in legacy path, include all
		}
		if r.RowID != "" && !r.Included {
			continue
		}
		out = append(out, Row{
			Instruction: r.Instruction,
			Input:       r.Input,
			Output:      r.Output,
		})
	}
	return out
}

// FilterSinceTimestamp drops rows before the cutoff.
func FilterSinceTimestamp(rows []PreviewRow, since time.Time) []PreviewRow {
	if since.IsZero() {
		return rows
	}
	out := make([]PreviewRow, 0, len(rows))
	for _, r := range rows {
		if r.MessageAt == "" {
			out = append(out, r)
			continue
		}
		ts, err := time.Parse(time.RFC3339, r.MessageAt)
		if err != nil || !ts.Before(since) {
			out = append(out, r)
		}
	}
	return out
}

// DeltaCount returns rows exported after since time.
func DeltaCount(rows []PreviewRow, since time.Time) int {
	if since.IsZero() {
		return len(rows)
	}
	n := 0
	for _, r := range rows {
		if r.MessageAt == "" {
			n++
			continue
		}
		ts, err := time.Parse(time.RFC3339, r.MessageAt)
		if err == nil && ts.After(since) {
			n++
		}
	}
	return n
}

func mergeExtraRows(rows []PreviewRow, extra []Row) []PreviewRow {
	if len(extra) == 0 {
		return rows
	}
	merged := make([]PreviewRow, 0, len(extra)+len(rows))
	for _, e := range extra {
		sk := strings.TrimSpace(e.SourceKind)
		if sk == "" {
			sk = "import"
		}
		ref := strings.TrimSpace(e.SourceRef)
		rid := strings.TrimSpace(e.RowID)
		if rid == "" {
			rid = rowID(sk, ref, e.Instruction, e.Output)
		}
		merged = append(merged, PreviewRow{Row: Row{
			RowID:       rid,
			Instruction: e.Instruction,
			Input:       e.Input,
			Output:      e.Output,
			SourceKind:  sk,
			SourceRef:   ref,
			Included:    true,
		}})
	}
	merged = append(merged, rows...)
	return merged
}

// AppendExtraRows merges learning rows and user-supplied rows for export.
func AppendExtraRows(learning []Row, extra []Row) []Row {
	if len(learning) == 0 && len(extra) == 0 {
		return nil
	}
	out := make([]Row, 0, len(learning)+len(extra))
	for _, e := range learning {
		r := e
		if strings.TrimSpace(r.SourceKind) == "" {
			r.SourceKind = "learning"
		}
		out = append(out, r)
	}
	out = append(out, extra...)
	return out
}

func validateMinRows(n int) error {
	if n < MinRows {
		return fmt.Errorf("only %d training rows (minimum %d)", n, MinRows)
	}
	return nil
}
