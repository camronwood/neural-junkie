package slack

import (
	"os"
	"strings"
)

// InboundDebugEnabled logs Slack Events API and inbound drop reasons when true.
func InboundDebugEnabled() bool {
	v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_DEBUG"))
	return v == "1" || strings.EqualFold(v, "true")
}
