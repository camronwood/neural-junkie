package protocol

import (
	"encoding/json"
	"strings"
)

// Edit-apply stream metadata keys (stream_delta / stream_end).
// Streams the resolved new_content progressively — not raw model tokens.
const (
	MetaStreamKind        = "stream_kind"
	StreamKindEditApply   = "edit_apply"
	MetaEditApplyChangeID = "change_id"
	MetaEditApplyPath     = "path"
	MetaEditApplyOffset   = "offset"
	MetaEditApplyHunkID   = "hunk_id"
	MetaEditApplyDone     = "edit_apply_done"
)

// Default chunk size for EmitEditApplyStream (bytes of UTF-8 text).
const EditApplyStreamChunkSize = 256

// EditApplyStreamMeta describes a progressive resolved-edit payload.
type EditApplyStreamMeta struct {
	StreamKind string `json:"stream_kind"`
	ChangeID   string `json:"change_id"`
	Path       string `json:"path"`
	Offset     int    `json:"offset"`
	HunkID     string `json:"hunk_id,omitempty"`
	Done       bool   `json:"edit_apply_done,omitempty"`
}

// ParseEditApplyStreamMeta extracts edit-apply stream fields from message metadata.
func ParseEditApplyStreamMeta(raw interface{}) (EditApplyStreamMeta, bool) {
	var meta EditApplyStreamMeta
	if raw == nil {
		return meta, false
	}
	switch m := raw.(type) {
	case map[string]interface{}:
		kind, _ := m[MetaStreamKind].(string)
		if strings.TrimSpace(kind) != StreamKindEditApply {
			return meta, false
		}
		meta.StreamKind = StreamKindEditApply
		meta.ChangeID, _ = m[MetaEditApplyChangeID].(string)
		meta.Path, _ = m[MetaEditApplyPath].(string)
		switch o := m[MetaEditApplyOffset].(type) {
		case float64:
			meta.Offset = int(o)
		case int:
			meta.Offset = o
		case json.Number:
			if n, err := o.Int64(); err == nil {
				meta.Offset = int(n)
			}
		}
		meta.HunkID, _ = m[MetaEditApplyHunkID].(string)
		if done, ok := m[MetaEditApplyDone].(bool); ok {
			meta.Done = done
		}
		meta.ChangeID = strings.TrimSpace(meta.ChangeID)
		meta.Path = strings.TrimSpace(meta.Path)
		return meta, meta.ChangeID != "" && meta.Path != ""
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return meta, false
		}
		var asMap map[string]interface{}
		if json.Unmarshal(data, &asMap) != nil {
			return meta, false
		}
		return ParseEditApplyStreamMeta(asMap)
	}
}

// IsEditApplyStreamMeta reports whether metadata marks an edit-apply stream.
func IsEditApplyStreamMeta(metadata map[string]interface{}) bool {
	_, ok := ParseEditApplyStreamMeta(metadata)
	return ok
}

// ChunkEditApplyContent splits resolved new_content into progressive text deltas.
// Each chunk is at most chunkSize bytes (last may be shorter). Empty input yields nil.
func ChunkEditApplyContent(newContent string, chunkSize int) []string {
	if newContent == "" {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = EditApplyStreamChunkSize
	}
	runes := []rune(newContent)
	if len(runes) == 0 {
		return nil
	}
	// Prefer rune-safe chunks while roughly respecting byte budget.
	out := make([]string, 0, (len(newContent)/chunkSize)+1)
	var b strings.Builder
	b.Grow(chunkSize)
	byteLen := 0
	for _, r := range runes {
		n := len(string(r))
		if b.Len() > 0 && byteLen+n > chunkSize {
			out = append(out, b.String())
			b.Reset()
			byteLen = 0
		}
		b.WriteRune(r)
		byteLen += n
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// ApplyEditApplyStreamMeta writes edit-apply fields onto a metadata map.
func ApplyEditApplyStreamMeta(dst map[string]interface{}, meta EditApplyStreamMeta) {
	if dst == nil {
		return
	}
	dst[MetaStreamKind] = StreamKindEditApply
	dst[MetaEditApplyChangeID] = strings.TrimSpace(meta.ChangeID)
	dst[MetaEditApplyPath] = strings.TrimSpace(meta.Path)
	dst[MetaEditApplyOffset] = meta.Offset
	if h := strings.TrimSpace(meta.HunkID); h != "" {
		dst[MetaEditApplyHunkID] = h
	}
	if meta.Done {
		dst[MetaEditApplyDone] = true
	}
}
