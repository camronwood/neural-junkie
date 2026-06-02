package slack

import (
	"strings"
)

// ForwardMatch is a rule match with source context for inbox forwarding.
type ForwardMatch struct {
	RuleType          ForwardRuleType
	RuleID            string
	SourceChannelID   string
	SourceChannelName string
	SourceTS          string
	SourceThreadTS    string
	SourceAuthor      string
	Permalink         string
	StrippedText      string
}

// EvaluateForwardRules returns the first matching forward rule for a channel message.
func EvaluateForwardRules(in InboundInput, rules []ForwardRule, ownerUserID string) (*ForwardMatch, bool) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, false
	}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if !channelMatchesRule(in.ChannelID, rule.SlackChannelIDs) {
			continue
		}
		switch rule.Type {
		case ForwardRuleMentionOfMe:
			if strings.Contains(in.Text, "<@"+ownerUserID+">") {
				return &ForwardMatch{
					RuleType:        ForwardRuleMentionOfMe,
					RuleID:          rule.ID,
					SourceChannelID: in.ChannelID,
					SourceTS:        in.SlackTS,
					SourceThreadTS:  in.ThreadTS,
					StrippedText:    StripSlackMentionMarkup(in.Text),
				}, true
			}
		case ForwardRulePrefix:
			prefix := rule.Prefix
			if prefix == "" {
				prefix = "nj:"
			}
			text := strings.TrimSpace(in.Text)
			if strings.HasPrefix(strings.ToLower(text), strings.ToLower(prefix)) {
				stripped := strings.TrimSpace(text[len(prefix):])
				if stripped == "" {
					continue
				}
				return &ForwardMatch{
					RuleType:        ForwardRulePrefix,
					RuleID:          rule.ID,
					SourceChannelID: in.ChannelID,
					SourceTS:        in.SlackTS,
					SourceThreadTS:  in.ThreadTS,
					StrippedText:    stripped,
				}, true
			}
		case ForwardRuleReaction:
			// Reaction matches are handled via handleReactionAdded, not message events.
		}
	}
	return nil, false
}

func channelMatchesRule(channelID string, watchlist []string) bool {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return false
	}
	if len(watchlist) == 0 {
		return false
	}
	for _, id := range watchlist {
		id = strings.TrimSpace(id)
		if id == "*" || id == channelID {
			return true
		}
	}
	return false
}

// ReactionForwardMatch builds a forward match from a reaction event on a channel message.
func ReactionForwardMatch(rule ForwardRule, channelID, messageTS, threadTS, text, author, channelName, permalink string) *ForwardMatch {
	return &ForwardMatch{
		RuleType:          ForwardRuleReaction,
		RuleID:            rule.ID,
		SourceChannelID:   channelID,
		SourceChannelName: channelName,
		SourceTS:          messageTS,
		SourceThreadTS:    threadTS,
		SourceAuthor:      author,
		Permalink:         permalink,
		StrippedText:      strings.TrimSpace(text),
	}
}
