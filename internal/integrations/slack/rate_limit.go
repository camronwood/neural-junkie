package slack

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	slackapi "github.com/slack-go/slack"
)

var slackRetryAfterRE = regexp.MustCompile(`(?i)retry after (\d+)`)

func slackRateLimitBackoff(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	var rateErr *slackapi.RateLimitedError
	if errors.As(err, &rateErr) && rateErr.RetryAfter > 0 {
		return rateErr.RetryAfter, true
	}
	msg := err.Error()
	if m := slackRetryAfterRE.FindStringSubmatch(msg); len(m) == 2 {
		if sec, convErr := strconv.Atoi(m[1]); convErr == nil && sec > 0 {
			return time.Duration(sec) * time.Second, true
		}
	}
	return 0, false
}

func isSlackRateLimited(err error) bool {
	_, ok := slackRateLimitBackoff(err)
	return ok
}

func isMissingScopeErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "missing_scope")
}

func isChannelNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "channel_not_found")
}
