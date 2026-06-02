package slack

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateChannelRequiresID(t *testing.T) {
	err := ValidateChannel(nil, "")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("err=%v", err)
	}
}

func TestShouldFallbackChannelList(t *testing.T) {
	if shouldFallbackChannelList(nil) {
		t.Fatal("nil err should not fallback")
	}
	if !shouldFallbackChannelList(errors.New("missing_scope")) {
		t.Fatal("missing_scope should fallback")
	}
	if !shouldFallbackChannelList(errors.New("not_allowed_token_type")) {
		t.Fatal("not_allowed_token_type should fallback")
	}
	if shouldFallbackChannelList(errors.New("invalid_auth")) {
		t.Fatal("invalid_auth should not fallback")
	}
}
