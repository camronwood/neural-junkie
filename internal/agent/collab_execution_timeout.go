package agent

import (
	"context"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// collabGenerationContext applies a deadline for collaboration task generation when requested.
func collabGenerationContext(ctx context.Context, msg *protocol.Message) (context.Context, context.CancelFunc) {
	cancel := context.CancelFunc(func() {})
	if msg == nil || msg.Type != protocol.MessageTypeCollabTask {
		return ctx, cancel
	}
	sec := executionTimeoutFromMetadata(msg.Metadata)
	if sec <= 0 {
		return ctx, cancel
	}
	return context.WithTimeout(ctx, time.Duration(sec)*time.Second)
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
