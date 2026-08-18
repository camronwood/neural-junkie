package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type streamMessageIDKey struct{}

func WithStreamMessageID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, streamMessageIDKey{}, id)
}

func StreamMessageIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(streamMessageIDKey{}).(string)
	return id
}

func (a *Agent) broadcastToolStep(ctx context.Context, msg *protocol.Message, streamMsgID string, ev ai.ToolStepEvent) {
	if a.Hub == nil || msg == nil {
		return
	}
	label := fmt.Sprintf("[%s] %s", ev.Name, ev.Kind)
	if ev.Preview != "" {
		label += ": " + ev.Preview
	}
	delta := protocol.NewMessage(protocol.MessageTypeStreamDelta, msg.Channel, a.Info, "")
	delta.ID = streamMsgID
	delta.ReplyTo = msg.ID
	if delta.Metadata == nil {
		delta.Metadata = make(map[string]interface{})
	}
	delta.Metadata["tool_step"] = ev.Kind
	delta.Metadata["tool_name"] = ev.Name
	delta.Metadata["tool_iteration"] = ev.Iteration
	delta.Metadata["tool_max_iterations"] = ev.MaxIterations
	delta.Metadata["tool_preview"] = label
	if msg.IsInThread() {
		delta.ThreadID = msg.ThreadID
		delta.IsThreadReply = true
	}
	a.Hub.BroadcastDirect(msg.Channel, delta)
	if ev.Kind == "start" || ev.Kind == "result" || ev.Kind == "error" || ev.Kind == "done" {
		a.sendThinkingActivity(msg, protocol.ThinkingActivityUsingTool, toolActivityDetail(ev))
		a.sendToolTelemetryEvent(msg, ev)
	}
}

// broadcastImplementationProgress updates the typing indicator during long implementation sessions.
// Does not emit stream_delta — interim deltas shared the final streamMsgID and left a stuck partial bubble in the UI.
func (a *Agent) broadcastImplementationProgress(msg *protocol.Message, _streamMsgID, text string) {
	if a == nil || msg == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	a.sendThinkingActivity(msg, protocol.ThinkingActivityImplementation, text)
}

// broadcastStreamEnd finalizes an in-progress stream so the desktop removes streamingMessages[id].
func (a *Agent) broadcastStreamEnd(msg *protocol.Message, streamMsgID string) {
	if a == nil || a.Hub == nil || msg == nil || streamMsgID == "" {
		return
	}
	endMsg := protocol.NewMessage(protocol.MessageTypeStreamEnd, msg.Channel, a.Info, "")
	endMsg.ID = streamMsgID
	endMsg.ReplyTo = msg.ID
	if msg.IsInThread() {
		endMsg.ThreadID = msg.ThreadID
		endMsg.IsThreadReply = true
	}
	a.Hub.BroadcastDirect(msg.Channel, endMsg)
}

func (a *Agent) streamTextAsTokens(ctx context.Context, msg *protocol.Message, streamMsgID, text string, terminalOut *streamTerminalCapture) (string, string, string, error) {
	tokenCh := make(chan ai.StreamToken, 32)
	go func() {
		defer close(tokenCh)
		words := strings.Fields(text)
		if len(words) == 0 {
			tokenCh <- ai.StreamToken{Content: text}
		} else {
			var b strings.Builder
			for i, w := range words {
				if i > 0 {
					b.WriteString(" ")
				}
				b.WriteString(w)
				if i%8 == 7 || i == len(words)-1 {
					tokenCh <- ai.StreamToken{Content: b.String() + " "}
					b.Reset()
				}
			}
		}
		tokenCh <- ai.StreamToken{Done: true, TerminalReason: protocol.TerminalReasonStop}
	}()
	return a.collectStreamTokens(ctx, msg, streamMsgID, tokenCh, terminalOut)
}

func (a *Agent) hasWorkspaceTools() bool {
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return false
	}
	return mcpServer.GetTool("read_file") != nil
}
