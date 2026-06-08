package slack

import (
	"errors"
	"testing"
	"time"

	slackapi "github.com/slack-go/slack"
)

func TestSlackRateLimitBackoff(t *testing.T) {
	wait, ok := slackRateLimitBackoff(&slackapi.RateLimitedError{RetryAfter: 30 * time.Second})
	if !ok || wait != 30*time.Second {
		t.Fatalf("RateLimitedError: got %v ok=%v", wait, ok)
	}

	wait, ok = slackRateLimitBackoff(errors.New("slack rate limit exceeded, retry after 30s"))
	if !ok || wait != 30*time.Second {
		t.Fatalf("message parse: got %v ok=%v", wait, ok)
	}

	_, ok = slackRateLimitBackoff(errors.New("network timeout"))
	if ok {
		t.Fatal("expected false for unrelated error")
	}
}
