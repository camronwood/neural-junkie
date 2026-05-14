package ai

import (
	"encoding/base64"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// openAITurnContent returns OpenAI Chat Completions `content` value: plain string or multimodal parts.
func openAITurnContent(userText string, images []protocol.UserImagePart) interface{} {
	if len(images) == 0 {
		return userText
	}
	var parts []map[string]interface{}
	if strings.TrimSpace(userText) != "" {
		parts = append(parts, map[string]interface{}{
			"type": "text",
			"text": userText,
		})
	}
	for _, im := range images {
		b64 := base64.StdEncoding.EncodeToString(im.Data)
		url := "data:" + im.MIME + ";base64," + b64
		parts = append(parts, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": url,
			},
		})
	}
	return parts
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

func buildOpenAIChatMessages(systemPrompt, userMessage string, conversationHistory []protocol.Message, currentImages []protocol.UserImagePart) []OpenAICompatibleMessage {
	messages := []OpenAICompatibleMessage{}
	if systemPrompt != "" {
		messages = append(messages, OpenAICompatibleMessage{Role: "system", Content: systemPrompt})
	}
	historyLimit := 10
	if len(conversationHistory) > historyLimit {
		conversationHistory = conversationHistory[len(conversationHistory)-historyLimit:]
	}
	for _, msg := range conversationHistory {
		role := "user"
		if msg.From.Type != protocol.AgentTypeGeneral {
			role = "assistant"
		}
		messages = append(messages, OpenAICompatibleMessage{Role: role, Content: msg.Content})
	}
	lastContent := openAITurnContent(userMessage, currentImages)
	messages = append(messages, OpenAICompatibleMessage{Role: "user", Content: lastContent})
	return messages
}
