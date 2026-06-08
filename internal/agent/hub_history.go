package agent

import (
	"fmt"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const agentHistoryBootstrapLimit = 50

type mergedHistoryHub interface {
	GetChannelMessagesMerged(channelName string, limit int) ([]*protocol.Message, error)
}

// bootstrapChannelHistory loads merged SQLite+memory tail for agent context.
func (a *Agent) bootstrapChannelHistory(channel string) ([]*protocol.Message, error) {
	if a == nil || a.Hub == nil {
		return nil, fmt.Errorf("no hub")
	}
	if mh, ok := a.Hub.(mergedHistoryHub); ok {
		return mh.GetChannelMessagesMerged(channel, agentHistoryBootstrapLimit)
	}
	return a.Hub.GetMessages(channel, 20)
}

func (a *Agent) historyForPriorReference(channel string) []*protocol.Message {
	local := a.channelHistory(channel)
	if mh, ok := a.Hub.(mergedHistoryHub); ok {
		merged, err := mh.GetChannelMessagesMerged(channel, 80)
		if err == nil && len(merged) > len(local) {
			return merged
		}
	}
	return local
}
