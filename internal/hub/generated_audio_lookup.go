package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// GeneratedAudioBytesFromMessage returns audio bytes and MIME from generated_audio metadata.
func GeneratedAudioBytesFromMessage(msg *protocol.Message) (mime string, data []byte, ok bool) {
	if msg == nil || msg.Metadata == nil {
		return "", nil, false
	}
	raw, ok := msg.Metadata["generated_audio"].(map[string]interface{})
	if !ok {
		return "", nil, false
	}
	mime, _ = raw["mime"].(string)
	mime = strings.TrimSpace(mime)
	if mime == "" {
		mime = "audio/wav"
	}
	if b64, _ := raw["data"].(string); strings.TrimSpace(b64) != "" {
		data, err := decodeGeneratedImageBase64(b64)
		if err == nil && len(data) > 0 {
			return mime, data, true
		}
	}
	if path, _ := raw["path"].(string); strings.TrimSpace(path) != "" {
		data, m, err := readGeneratedAudioFile(path)
		if err == nil && len(data) > 0 {
			if m != "" {
				mime = m
			}
			return mime, data, true
		}
	}
	if path, ok := generatedAudioFileForMessageID(msg.ID); ok {
		data, m, err := readGeneratedAudioFile(path)
		if err == nil && len(data) > 0 {
			if m != "" {
				mime = m
			}
			return mime, data, true
		}
	}
	return "", nil, false
}
