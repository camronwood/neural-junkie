package agent

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type captureAI struct {
	mu     sync.Mutex
	prompt string
}

func (c *captureAI) GenerateResponse(_ context.Context, prompt string, _ []protocol.Message) (string, error) {
	c.mu.Lock()
	c.prompt = prompt
	c.mu.Unlock()
	return "captured", nil
}
func (c *captureAI) GetModel() string { return "capture" }
func (c *captureAI) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	return c.GenerateResponse(ctx, prompt, conversationHistory)
}

type captureHub struct {
	sent []*protocol.Message
}

func (h *captureHub) SendMessage(msg *protocol.Message) error {
	h.sent = append(h.sent, msg)
	return nil
}
func (h *captureHub) Subscribe(string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (h *captureHub) GetMessages(string, int) ([]*protocol.Message, error) { return nil, nil }
func (h *captureHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (h *captureHub) GetThreadParentAuthor(string) string                     { return "" }
func (h *captureHub) GetCommandHandler() CommandHandlerInterface              { return nil }
func (h *captureHub) BroadcastDirect(string, *protocol.Message)                 {}
func (h *captureHub) GetAgentChannels(string) []string                          { return nil }
func (h *captureHub) GetChannelType(string) protocol.ChannelType                { return protocol.ChannelTypePublic }
func (h *captureHub) GetChannelSessionSummary(string) string                    { return "" }
func (h *captureHub) GetThreadMessages(string, int) ([]*protocol.Message, error) {
	return nil, nil
}
func (h *captureHub) IsChannelHeld(string) bool { return false }
func (h *captureHub) ImageGenerationEnabled() bool {
	return false
}
func (h *captureHub) GenerateAndPostImage(context.Context, string, protocol.AgentInfo, string, string) error {
	return nil
}

func TestRepoAgentInterceptorInjectsIndexIntoPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/README.md", []byte("# Widget\nA widget service."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/main.go", []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cap := &captureAI{}
	hub := &captureHub{}
	ra, err := NewRepoAgent("widget-expert", dir, cap, hub)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ra.StartWithIndexing(ctx, "general"); err != nil {
		t.Fatal(err)
	}

	waitDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(waitDeadline) {
		ra.mu.RLock()
		ready := ra.index != nil && !ra.isIndexing
		ra.mu.RUnlock()
		if ready {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	ra.mu.RLock()
	idxReady := ra.index != nil && !ra.isIndexing
	ra.mu.RUnlock()
	if !idxReady {
		t.Fatal("indexing did not complete")
	}

	question := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: protocol.AgentTypeGeneral},
		"@widget-expert what is this project?",
	)
	question.Mentions = []string{ra.Info.ID}

	ra.handleMessage(ctx, question)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cap.mu.Lock()
		p := cap.prompt
		cap.mu.Unlock()
		if strings.Contains(p, "Widget") || strings.Contains(p, "README") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cap.mu.Lock()
	prompt := cap.prompt
	cap.mu.Unlock()
	if prompt == "" {
		t.Fatal("expected GenerateResponse to be called with repo prompt")
	}
	if !strings.Contains(prompt, ai.SystemPromptSeparator) {
		t.Fatal("repo prompt must use system/user separator for Ollama")
	}
	system, user := ai.SplitSystemPrompt(prompt)
	if !strings.Contains(system, "Widget") && !strings.Contains(system, "README") {
		snippet := system
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		t.Fatalf("system prompt missing repo context: %q", snippet)
	}
	if !strings.Contains(user, "what is this project") {
		t.Fatal("user section should contain the question")
	}
}
