package hub

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimiterRoomJoinHasLowerLimit(t *testing.T) {
	rl := &RateLimiter{
		events:  make(map[string][]time.Time),
		enabled: true,
		readMax: 300,
		mutMax:  120,
		window:  time.Minute,
	}
	r, _ := http.NewRequest("POST", "http://example.com/api/room/join", nil)
	r.RemoteAddr = "1.2.3.4:1234"
	for i := 0; i < 20; i++ {
		if !rl.Allow(r) {
			t.Fatalf("expected allow at %d", i)
		}
	}
	if rl.Allow(r) {
		t.Fatalf("expected deny after 20")
	}
}

