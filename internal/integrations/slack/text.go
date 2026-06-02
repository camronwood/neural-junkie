package slack

import (
	"regexp"
	"strings"
)

var slackMentionMarkup = regexp.MustCompile(`<@[A-Z0-9]+>`)

// StripSlackMentionMarkup removes Slack user mention markup (<@U…>) from message text.
func StripSlackMentionMarkup(text string) string {
	stripped := slackMentionMarkup.ReplaceAllString(text, "")
	return strings.TrimSpace(strings.Join(strings.Fields(stripped), " "))
}
