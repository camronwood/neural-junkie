package ai

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func userMsg(content string) protocol.Message {
	return protocol.Message{
		Type:    protocol.MessageTypeChat,
		From:    protocol.AgentInfo{ID: "user-1", Name: "User", Type: "human"},
		Content: content,
	}
}

func assistantMsg(content string) protocol.Message {
	return protocol.Message{
		Type:    protocol.MessageTypeChat,
		From:    protocol.AgentInfo{ID: "agent-1", Name: "Bot", Type: "assistant"},
		Content: content,
	}
}

func TestBuildOpenAIChatMessages_preservesSystemAndLatestUser(t *testing.T) {
	msgs, warnings := buildOpenAIChatMessages("sys", "latest", []protocol.Message{
		userMsg("old"),
		assistantMsg("reply"),
	}, nil)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(msgs) < 2 {
		t.Fatalf("got %d messages", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "sys" {
		t.Fatalf("system = %#v", msgs[0])
	}
	last := msgs[len(msgs)-1]
	if last.Role != "user" || last.Content != "latest" {
		t.Fatalf("latest = %#v", last)
	}
}

func TestBuildOpenAIChatMessages_historyBudgetCutoff(t *testing.T) {
	// Three large prior turns (~5 KiB each). With a 12 KiB history budget,
	// newest two should fit; oldest should drop.
	big := strings.Repeat("x", 5*1024)
	history := []protocol.Message{
		userMsg("OLD-" + big),
		assistantMsg("MID-" + big),
		userMsg("NEW-" + big),
	}
	msgs, _ := buildOpenAIChatMessages("sys", "ask", history, nil)

	var historyContents []string
	for _, m := range msgs {
		if m.Role == "system" || (m.Role == "user" && m.Content == "ask") {
			continue
		}
		s, ok := m.Content.(string)
		if !ok {
			t.Fatalf("history content type %T", m.Content)
		}
		historyContents = append(historyContents, s[:4])
	}
	if len(historyContents) != 2 {
		t.Fatalf("history turns = %v, want 2 newest", historyContents)
	}
	if historyContents[0] != "MID-" || historyContents[1] != "NEW-" {
		t.Fatalf("history order/prefix = %v, want MID then NEW", historyContents)
	}
}

func TestBuildOpenAIChatMessages_historyMessageCap(t *testing.T) {
	SetHubRuntimeOptions(config.PerformanceConfig{}, config.OllamaConfig{})
	history := make([]protocol.Message, 15)
	for i := range history {
		history[i] = userMsg("m" + strings.Repeat("n", i%3))
	}

	msgs, _ := buildOpenAIChatMessages("", "ask", history, nil)
	historyCount := 0
	for _, m := range msgs {
		if m.Role == "user" && m.Content == "ask" {
			continue
		}
		historyCount++
	}
	want := MaxHistoryMessages()
	if historyCount != want {
		t.Fatalf("historyCount=%d, want %d", historyCount, want)
	}
}

func TestBuildOpenAIChatMessages_imageGuardSubstitution(t *testing.T) {
	t.Setenv("NJ_OPENAI_INLINE_IMAGE_MAX_BYTES", "64")
	small := protocol.UserImagePart{MIME: "image/png", Data: []byte("tiny-ok")}
	large := protocol.UserImagePart{MIME: "image/jpeg", Data: make([]byte, 128)}

	msgs, warnings := buildOpenAIChatMessages("sys", "see this", nil, []protocol.UserImagePart{small, large})
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1", warnings)
	}
	if !strings.Contains(warnings[0], "IMAGE_TOO_LARGE") {
		t.Fatalf("warning = %q", warnings[0])
	}
	last := msgs[len(msgs)-1]
	parts, ok := last.Content.([]map[string]interface{})
	if !ok {
		t.Fatalf("content type %T", last.Content)
	}
	var sawImageURL, sawPlaceholder bool
	for _, p := range parts {
		switch p["type"] {
		case "image_url":
			sawImageURL = true
		case "text":
			text, _ := p["text"].(string)
			if strings.Contains(text, "IMAGE_TOO_LARGE") {
				sawPlaceholder = true
			}
		}
	}
	if !sawImageURL {
		t.Fatal("expected small image to remain inlined")
	}
	if !sawPlaceholder {
		t.Fatal("expected large image placeholder text")
	}
}

func TestOpenAIInlineImageMaxBytes_envOverride(t *testing.T) {
	t.Setenv("NJ_OPENAI_INLINE_IMAGE_MAX_BYTES", "1024")
	if got := openAIInlineImageMaxBytes(); got != 1024 {
		t.Fatalf("got %d", got)
	}
	t.Setenv("NJ_OPENAI_INLINE_IMAGE_MAX_BYTES", "bogus")
	if got := openAIInlineImageMaxBytes(); got != defaultOpenAIInlineImageMaxBytes {
		t.Fatalf("bogus env got %d", got)
	}
	t.Setenv("NJ_OPENAI_INLINE_IMAGE_MAX_BYTES", "")
	if got := openAIInlineImageMaxBytes(); got != defaultOpenAIInlineImageMaxBytes {
		t.Fatalf("empty env got %d", got)
	}
}

func TestTrimOpenAIHistory_pinsNewestWhenOverBudget(t *testing.T) {
	huge := strings.Repeat("z", openAIHistoryBodyBudgetBytes+100)
	history := []protocol.Message{
		userMsg("small"),
		userMsg(huge),
	}
	got := trimOpenAIHistory(history, openAIHistoryBodyBudgetBytes, 10)
	if len(got) != 1 {
		t.Fatalf("got %d msgs, want newest only", len(got))
	}
	if got[0].Content != huge {
		t.Fatal("expected newest oversized turn to be pinned")
	}
}
