package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Client metadata keys for user-attached images (desktop + hub).
const (
	MetadataUserImages = "user_images"
	// Legacy single-image keys (still accepted by ExtractUserImages).
	MetadataImageData = "image_data"
	MetadataImageType = "image_type"
	// Placeholder after server redaction for WebSocket / public API.
	MetadataImageDataRedacted = "image_data_redacted"
)

const (
	MaxUserImageCount       = 6
	MaxUserImageBytesEach   = 5 * 1024 * 1024
	MaxUserImagesTotalBytes = 12 * 1024 * 1024
)

// UserImagePart is decoded image bytes + MIME for provider APIs.
type UserImagePart struct {
	MIME string
	Data []byte
}

// ExtractUserImages reads canonical user_images[] and legacy image_data/image_type from message metadata.
func ExtractUserImages(msg *Message) []UserImagePart {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	var out []UserImagePart
	total := 0

	if raw, ok := msg.Metadata[MetadataUserImages]; ok && raw != nil {
		if arr, ok := raw.([]interface{}); ok {
			for _, item := range arr {
				if len(out) >= MaxUserImageCount {
					break
				}
				fm, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				mime, _ := fm["mime"].(string)
				if mime == "" {
					mime = "image/png"
				}
				dataRaw, ok := fm["data"]
				if !ok {
					continue
				}
				data, err := decodeFlexibleBase64(dataRaw)
				if err != nil || len(data) == 0 {
					continue
				}
				if len(data) > MaxUserImageBytesEach {
					data = data[:MaxUserImageBytesEach]
				}
				if total+len(data) > MaxUserImagesTotalBytes {
					break
				}
				total += len(data)
				out = append(out, UserImagePart{MIME: mime, Data: data})
			}
		}
	}

	if len(out) == 0 {
		if raw, ok := msg.Metadata[MetadataImageData]; ok {
			mime, _ := msg.Metadata[MetadataImageType].(string)
			if mime == "" {
				mime = "image/png"
			}
			if data, err := decodeFlexibleBase64(raw); err == nil && len(data) > 0 {
				if len(data) > MaxUserImageBytesEach {
					data = data[:MaxUserImageBytesEach]
				}
				out = append(out, UserImagePart{MIME: mime, Data: data})
			}
		}
	}

	return out
}

func decodeFlexibleBase64(v interface{}) ([]byte, error) {
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil, fmt.Errorf("empty")
		}
		// Strip data URL prefix if present
		if i := strings.Index(s, ","); i >= 0 && strings.HasPrefix(strings.ToLower(s), "data:") {
			s = s[i+1:]
		}
		// JSON from browser is standard base64; also accept raw bytes interpreted as string
		dec, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			dec, err = base64.RawStdEncoding.DecodeString(s)
		}
		if err != nil {
			return nil, err
		}
		return dec, nil
	case []byte:
		return append([]byte(nil), x...), nil
	case json.Number:
		return nil, fmt.Errorf("unsupported number")
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// MessageHasUserImages returns true if metadata may contain decodable user images.
func MessageHasUserImages(msg *Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	if _, ok := msg.Metadata[MetadataUserImages]; ok {
		return true
	}
	if _, ok := msg.Metadata[MetadataImageData]; ok {
		return true
	}
	return false
}

// SanitizeUserImagesMetadata trims image payloads in-place on msg.Metadata to server limits
// and normalizes to MetadataUserImages when possible.
func SanitizeUserImagesMetadata(msg *Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	parts := ExtractUserImages(msg)
	if len(parts) == 0 {
		// Drop unusable oversized string keys to save memory
		if raw, ok := msg.Metadata[MetadataImageData]; ok {
			if s, ok := raw.(string); ok && len(s) > MaxUserImageBytesEach*2 {
				delete(msg.Metadata, MetadataImageData)
				delete(msg.Metadata, MetadataImageType)
			}
		}
		return
	}
	arr := make([]interface{}, 0, len(parts))
	total := 0
	for _, p := range parts {
		if total+len(p.Data) > MaxUserImagesTotalBytes {
			break
		}
		total += len(p.Data)
		b64 := base64.StdEncoding.EncodeToString(p.Data)
		arr = append(arr, map[string]interface{}{
			"mime": p.MIME,
			"data": b64,
		})
	}
	msg.Metadata[MetadataUserImages] = arr
	delete(msg.Metadata, MetadataImageData)
	delete(msg.Metadata, MetadataImageType)
}

// CloneMessage returns a deep copy via JSON round-trip (adequate for protocol.Message).
func CloneMessage(m *Message) (*Message, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var c Message
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// RedactImageBinaryMetadata replaces large image blobs in metadata with lightweight placeholders.
func RedactImageBinaryMetadata(msg *Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	if raw, ok := msg.Metadata[MetadataUserImages]; ok && raw != nil {
		if arr, ok := raw.([]interface{}); ok {
			red := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				fm, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				mime, _ := fm["mime"].(string)
				dataStr, _ := fm["data"].(string)
				red = append(red, map[string]interface{}{
					"mime":         mime,
					"redacted":     true,
					"approx_bytes": len(dataStr),
				})
			}
			msg.Metadata[MetadataUserImages] = red
		}
	}
	if _, ok := msg.Metadata[MetadataImageData]; ok {
		msg.Metadata[MetadataImageDataRedacted] = true
		delete(msg.Metadata, MetadataImageData)
	}
	if raw, ok := msg.Metadata["generated_image"].(map[string]interface{}); ok {
		if _, has := raw["data"]; has {
			m := map[string]interface{}{}
			for k, v := range raw {
				if k == "data" {
					m["data_redacted"] = true
					if s, ok := v.(string); ok {
						m["approx_bytes"] = len(s)
					}
					continue
				}
				m[k] = v
			}
			msg.Metadata["generated_image"] = m
		}
	}
}
