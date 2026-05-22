package slack

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// FormatSlackChannelDisplayName returns a sidebar-safe label such as "#cursor-test".
func FormatSlackChannelDisplayName(slackChannelName string) string {
	name := strings.TrimSpace(strings.TrimPrefix(slackChannelName, "#"))
	if name == "" {
		return ""
	}
	return "#" + name
}

// SlackChannelDescription is stored on the hub channel for tooltips and legacy clients.
func SlackChannelDescription(slackChannelName string) string {
	label := FormatSlackChannelDisplayName(slackChannelName)
	if label == "" {
		return "Slack bridge"
	}
	return "Slack: " + label
}

// SyncChannelDisplay updates hub channel display_name and description from a binding.
func SyncChannelDisplay(hub HubClient, b Binding) {
	if hub == nil || b.NJChannel == "" {
		return
	}
	label := FormatSlackChannelDisplayName(b.SlackChannelName)
	if label == "" {
		return
	}
	_ = hub.SetChannelDisplay(b.NJChannel, label, SlackChannelDescription(b.SlackChannelName))
}

// EnrichChannelsFromBindings sets display_name on slack:* hub channels when bindings carry names.
func EnrichChannelsFromBindings(channels []*protocol.Channel, store *BindingStore) {
	if store == nil || len(channels) == 0 {
		return
	}
	byNJ := make(map[string]Binding, len(store.items))
	for _, b := range store.List() {
		if !b.Enabled || b.NJChannel == "" {
			continue
		}
		byNJ[b.NJChannel] = b
	}
	for _, ch := range channels {
		if ch == nil || !strings.HasPrefix(ch.Name, "slack:") {
			continue
		}
		b, ok := byNJ[ch.Name]
		if !ok {
			continue
		}
		label := FormatSlackChannelDisplayName(b.SlackChannelName)
		if label == "" {
			continue
		}
		ch.DisplayName = label
		if strings.TrimSpace(ch.Description) == "" || ch.Description == "Slack bridge" {
			ch.Description = SlackChannelDescription(b.SlackChannelName)
		}
	}
}
