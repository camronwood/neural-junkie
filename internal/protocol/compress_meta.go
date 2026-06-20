package protocol

// Context compression metadata keys on agent response messages.
const (
	MetadataCompressBytesIn  = "context_compress_bytes_in"
	MetadataCompressBytesOut = "context_compress_bytes_out"
	MetadataCompressStrategy = "context_compress_strategy"
	MetadataCompressRefs     = "context_compress_refs"
)

// CompressMeta captures per-turn compression stats for UI display.
type CompressMeta struct {
	BytesIn   int    `json:"bytes_in,omitempty"`
	BytesOut  int    `json:"bytes_out,omitempty"`
	Strategy  string `json:"strategy,omitempty"`
	Refs      string `json:"refs,omitempty"`
}

// ApplyCompressMeta writes compression fields onto message metadata (accumulates bytes).
func ApplyCompressMeta(msg *Message, meta CompressMeta) {
	if msg == nil {
		return
	}
	if meta.BytesIn == 0 && meta.BytesOut == 0 && meta.Strategy == "" && meta.Refs == "" {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	if meta.BytesIn > 0 {
		prev, _ := msg.Metadata[MetadataCompressBytesIn].(float64)
		if prevInt, ok := msg.Metadata[MetadataCompressBytesIn].(int); ok {
			prev = float64(prevInt)
		}
		msg.Metadata[MetadataCompressBytesIn] = int(prev) + meta.BytesIn
	}
	if meta.BytesOut > 0 {
		prev, _ := msg.Metadata[MetadataCompressBytesOut].(float64)
		if prevInt, ok := msg.Metadata[MetadataCompressBytesOut].(int); ok {
			prev = float64(prevInt)
		}
		msg.Metadata[MetadataCompressBytesOut] = int(prev) + meta.BytesOut
	}
	if meta.Strategy != "" {
		msg.Metadata[MetadataCompressStrategy] = meta.Strategy
	}
	if meta.Refs != "" {
		existing, _ := msg.Metadata[MetadataCompressRefs].(string)
		if existing == "" {
			msg.Metadata[MetadataCompressRefs] = meta.Refs
		} else {
			msg.Metadata[MetadataCompressRefs] = existing + "," + meta.Refs
		}
	}
}

// ExtractCompressMeta reads compression metadata from a message.
func ExtractCompressMeta(msg *Message) CompressMeta {
	if msg == nil || msg.Metadata == nil {
		return CompressMeta{}
	}
	out := CompressMeta{}
	if v, ok := msg.Metadata[MetadataCompressBytesIn].(float64); ok {
		out.BytesIn = int(v)
	} else if v, ok := msg.Metadata[MetadataCompressBytesIn].(int); ok {
		out.BytesIn = v
	}
	if v, ok := msg.Metadata[MetadataCompressBytesOut].(float64); ok {
		out.BytesOut = int(v)
	} else if v, ok := msg.Metadata[MetadataCompressBytesOut].(int); ok {
		out.BytesOut = v
	}
	if v, ok := msg.Metadata[MetadataCompressStrategy].(string); ok {
		out.Strategy = v
	}
	if v, ok := msg.Metadata[MetadataCompressRefs].(string); ok {
		out.Refs = v
	}
	return out
}
