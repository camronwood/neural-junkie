package slack

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	slackapi "github.com/slack-go/slack"
)

const maxSlackUploadBytes = 10 * 1024 * 1024

// GeneratedImagePayload is hub metadata for an inline or file-backed generated image.
type GeneratedImagePayload struct {
	MIME string
	Data []byte
	Path string
}

// ExtractGeneratedImage reads generated_image metadata or returns nil.
func ExtractGeneratedImage(msg *protocol.Message) *GeneratedImagePayload {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	raw, ok := msg.Metadata["generated_image"].(map[string]interface{})
	if !ok || raw == nil {
		return nil
	}
	out := &GeneratedImagePayload{}
	if p, ok := raw["path"].(string); ok {
		out.Path = strings.TrimSpace(p)
	}
	if d, ok := raw["data"].(string); ok && strings.TrimSpace(d) != "" {
		b, err := base64.StdEncoding.DecodeString(d)
		if err == nil && len(b) > 0 {
			out.Data = b
		}
	}
	if m, ok := raw["mime"].(string); ok {
		out.MIME = strings.TrimSpace(m)
	}
	if len(out.Data) == 0 && out.Path != "" {
		b, mime, err := readImageFile(out.Path)
		if err == nil {
			out.Data = b
			if out.MIME == "" {
				out.MIME = mime
			}
		}
	}
	if len(out.Data) == 0 {
		return nil
	}
	if len(out.Data) > maxSlackUploadBytes {
		log.Printf("[slack] generated image too large for upload (%d bytes), skipping file upload", len(out.Data))
		return nil
	}
	if out.MIME == "" {
		out.MIME = "image/png"
	}
	return out
}

func readImageFile(path string) ([]byte, string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, "", fmt.Errorf("empty path")
	}
	clean := filepath.Clean(path)
	b, err := os.ReadFile(clean)
	if err != nil {
		return nil, "", err
	}
	if len(b) > maxSlackUploadBytes {
		return nil, "", fmt.Errorf("file too large")
	}
	return b, mimeFromImagePath(clean), nil
}

func mimeFromImagePath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func filenameForMIME(mime string) string {
	switch strings.ToLower(mime) {
	case "image/jpeg", "image/jpg":
		return "generated.jpg"
	case "image/gif":
		return "generated.gif"
	case "image/webp":
		return "generated.webp"
	default:
		return "generated.png"
	}
}

// UploadGeneratedImage posts an image file to a Slack channel (optional thread).
func UploadGeneratedImage(api *slackapi.Client, channelID, threadTS, title string, img *GeneratedImagePayload) error {
	if api == nil || img == nil || len(img.Data) == 0 {
		return fmt.Errorf("missing slack client or image data")
	}
	if channelID == "" {
		return fmt.Errorf("empty slack channel id")
	}
	if title == "" {
		title = "Generated image"
	}
	params := slackapi.UploadFileParameters{
		Reader:          bytes.NewReader(img.Data),
		Filename:        filenameForMIME(img.MIME),
		FileSize:        len(img.Data),
		Title:           title,
		Channel:         channelID,
		ThreadTimestamp: threadTS,
	}
	_, err := api.UploadFile(params)
	return err
}
