package slack

import (
	"strings"
	"testing"
)

func TestValidateChannelRequiresID(t *testing.T) {
	err := ValidateChannel(nil, "")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("err=%v", err)
	}
}
