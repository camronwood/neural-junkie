package agent

import (
	"context"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	defaultCollabRecapTimeoutSeconds      = 240
	defaultCollabDiscussionTimeoutSeconds = 480
)

// collabGenerationContext applies a deadline for collaboration turns (tasks, recap, discussion).
func collabGenerationContext(ctx context.Context, msg *protocol.Message) (context.Context, context.CancelFunc) {
	cancel := context.CancelFunc(func() {})
	if msg == nil {
		return ctx, cancel
	}
	sec := collabGenerationTimeoutSeconds(msg)
	if sec <= 0 {
		return ctx, cancel
	}
	return context.WithTimeout(ctx, time.Duration(sec)*time.Second)
}

func collabGenerationTimeoutSeconds(msg *protocol.Message) int {
	if msg != nil && strings.TrimSpace(msg.Channel) == "implement-scenarios" {
		return int(implScenarioSessionFrontendTimeout / time.Second)
	}
	switch msg.Type {
	case protocol.MessageTypeCollabTask:
		if sec := executionTimeoutFromMetadata(msg.Metadata); sec > 0 {
			return sec
		}
		return collaboration.DefaultCollabFileExecutionTimeoutSeconds
	case protocol.MessageTypeCollabRecap:
		return defaultCollabRecapTimeoutSeconds
	case protocol.MessageTypeCollabDiscussion:
		switch msg.GetCollaborationPhase() {
		case "planning", "reviewing", "executing":
			return defaultCollabDiscussionTimeoutSeconds
		}
	}
	return 0
}

func executionTimeoutFromMetadata(md map[string]interface{}) int {
	if md == nil {
		return 0
	}
	switch v := md["execution_timeout_seconds"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
