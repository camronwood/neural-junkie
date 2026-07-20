package hub

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimiterRespectsReconfigureDisabled(t *testing.T) {
	rl := &RateLimiter{
		events:  make(map[string][]time.Time),
		enabled: true,
		readMax: 300,
		mutMax:  2,
		window:  time.Minute,
	}
	r, _ := http.NewRequest("POST", "http://example.com/api/send", nil)
	r.RemoteAddr = "1.2.3.4:1234"
	if !rl.Allow(r) || !rl.Allow(r) {
		t.Fatal("expected first two mutates allowed")
	}
	if rl.Allow(r) {
		t.Fatal("expected deny at mutate budget")
	}

	rl.Reconfigure(false, 300, 120)
	if !rl.Allow(r) {
		t.Fatal("expected allow after rate limit disabled via Reconfigure")
	}
}

func TestRateLimiterOpenDMExempt(t *testing.T) {
	rl := &RateLimiter{
		events:  make(map[string][]time.Time),
		enabled: true,
		readMax: 300,
		mutMax:  1,
		window:  time.Minute,
	}
	openDM, _ := http.NewRequest("POST", "http://example.com/api/channels/open-dm", nil)
	openDM.RemoteAddr = "1.2.3.4:1234"
	send, _ := http.NewRequest("POST", "http://example.com/api/send", nil)
	send.RemoteAddr = "1.2.3.4:1234"

	for i := 0; i < 5; i++ {
		if !rl.Allow(openDM) {
			t.Fatalf("open-dm should be exempt at %d", i)
		}
	}
	if !rl.Allow(send) {
		t.Fatal("send should still have full mutate budget after exempt open-dm calls")
	}
	if rl.Allow(send) {
		t.Fatal("second send should hit mutate budget of 1")
	}
}
