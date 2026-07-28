package ai

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// openAIHistoryBodyBudgetBytes matches maxBudgetHistoryBody in
// internal/agent/context_budget.go (12 KiB recent-turns section).
const openAIHistoryBodyBudgetBytes = 12 * 1024

// defaultOpenAIInlineImageMaxBytes is the raw (pre-base64) size limit for
// inlining images as data: URLs. Override with NJ_OPENAI_INLINE_IMAGE_MAX_BYTES.
const defaultOpenAIInlineImageMaxBytes = 32 * 1024

// openAIInlineImageMaxBytes returns the inline image byte threshold.
func openAIInlineImageMaxBytes() int {
	raw := strings.TrimSpace(os.Getenv("NJ_OPENAI_INLINE_IMAGE_MAX_BYTES"))
	if raw == "" {
		return defaultOpenAIInlineImageMaxBytes
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultOpenAIInlineImageMaxBytes
	}
	return n
}

// openAIHistoryBudgetBytes returns the byte budget for prior turns in the
// OpenAI messages array (system + current user turn are always preserved).
func openAIHistoryBudgetBytes() int {
	return openAIHistoryBodyBudgetBytes
}

// openAITurnContent returns OpenAI Chat Completions `content` value: plain string or multimodal parts.
// Oversized images are replaced with a text placeholder; warnings describe each omission.
func openAITurnContent(userText string, images []protocol.UserImagePart) (interface{}, []string) {
	if len(images) == 0 {
		return userText, nil
	}
	maxBytes := openAIInlineImageMaxBytes()
	var warnings []string
	var parts []map[string]interface{}
	if strings.TrimSpace(userText) != "" {
		parts = append(parts, map[string]interface{}{
			"type": "text",
			"text": userText,
		})
	}
	for i, im := range images {
		mime := strings.TrimSpace(im.MIME)
		if mime == "" {
			mime = "image/png"
		}
		if len(im.Data) > maxBytes {
			placeholder := fmt.Sprintf(
				"[IMAGE_TOO_LARGE: index=%d mime=%s bytes=%d limit=%d; not inlined — attach a smaller image or raise NJ_OPENAI_INLINE_IMAGE_MAX_BYTES]",
				i, mime, len(im.Data), maxBytes,
			)
			parts = append(parts, map[string]interface{}{
				"type": "text",
				"text": placeholder,
			})
			warnings = append(warnings, placeholder)
			continue
		}
		b64 := base64.StdEncoding.EncodeToString(im.Data)
		url := "data:" + mime + ";base64," + b64
		parts = append(parts, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": url,
			},
		})
	}
	if len(parts) == 0 {
		return userText, warnings
	}
	return parts, warnings
}

func openAIMessageTextContent(c interface{}) string {
	switch v := c.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, part := range v {
			m, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			if typ, _ := m["type"].(string); typ == "text" {
				if t, ok := m["text"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
		return sb.String()
	default:
		return ""
	}
}

func estimateOpenAIHistoryMsgBytes(msg protocol.Message) int {
	return len(strings.TrimSpace(msg.Content))
}

// trimOpenAIHistory keeps the newest turns that fit both the byte budget and
// MaxHistoryMessages(). Newest messages are preferred; an empty slice is fine.
func trimOpenAIHistory(history []protocol.Message, budgetBytes, maxMessages int) []protocol.Message {
	if len(history) == 0 {
		return nil
	}
	if budgetBytes <= 0 {
		budgetBytes = openAIHistoryBodyBudgetBytes
	}
	if maxMessages <= 0 {
		maxMessages = MaxHistoryMessages()
	}

	selected := make([]protocol.Message, 0, maxMessages)
	used := 0
	for i := len(history) - 1; i >= 0; i-- {
		if len(selected) >= maxMessages {
			break
		}
		msg := history[i]
		n := estimateOpenAIHistoryMsgBytes(msg)
		if n == 0 {
			continue
		}
		// Always keep at least the newest non-empty turn even if it alone
		// exceeds the budget (mirrors pinning the latest user turn).
		if used+n > budgetBytes && len(selected) > 0 {
			break
		}
		selected = append(selected, msg)
		used += n
	}
	// Reverse to chronological order.
	for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
		selected[i], selected[j] = selected[j], selected[i]
	}
	return selected
}

// buildOpenAIChatMessages assembles OpenAI-compatible chat messages.
// System prompt and the latest user turn are always preserved.
// Prior turns are trimmed by byte budget (aligned with agent context_budget
// history body) and MaxHistoryMessages().
// ImageGuardWarnings lists placeholders substituted for oversized inline images.
func buildOpenAIChatMessages(systemPrompt, userMessage string, conversationHistory []protocol.Message, currentImages []protocol.UserImagePart) (messages []OpenAICompatibleMessage, imageGuardWarnings []string) {
	if systemPrompt != "" {
		messages = append(messages, OpenAICompatibleMessage{Role: "system", Content: systemPrompt})
	}
	for _, msg := range trimOpenAIHistory(conversationHistory, openAIHistoryBudgetBytes(), MaxHistoryMessages()) {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		messages = append(messages, OpenAICompatibleMessage{
			Role:    ChatRoleForHistory(msg),
			Content: content,
		})
	}
	lastContent, warnings := openAITurnContent(userMessage, currentImages)
	imageGuardWarnings = warnings
	messages = append(messages, OpenAICompatibleMessage{Role: "user", Content: lastContent})
	return messages, imageGuardWarnings
}
