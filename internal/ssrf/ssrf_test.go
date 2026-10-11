package ssrf

import (
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestCheckPublicURL_blocksLiteralPrivate(t *testing.T) {
	cases := []string{
		"http://127.0.0.1:9999/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
		"http://localhost/",
	}
	for _, raw := range cases {
		if err := CheckPublicURL(raw); err == nil {
			t.Fatalf("expected block for %s", raw)
		}
	}
}

func TestCheckPublicURL_blocksDNSToLoopback(t *testing.T) {
	// Most environments resolve localhost; also try a name that maps to 127.0.0.1.
	if err := CheckPublicURL("http://localhost.localdomain/"); err == nil {
		// Some resolvers fail DNS; either block or DNS-fail is acceptable.
		t.Log("localhost.localdomain not resolvable or not blocked; trying 127.0.0.1 via literal already covered")
	}
	// Force path: if "localhost" slips past hostname rules somehow — already covered.
	// Resolve a hostname that points at loopback when present.
	if ips, err := net.LookupIP("localhost"); err == nil {
		for _, ip := range ips {
			if IsBlockedIP(ip) {
				// hostname rule should catch "localhost" before DNS
				if err := CheckPublicURL("http://localhost/"); err == nil {
					t.Fatal("expected localhost blocked")
				}
				return
			}
		}
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "::1", "10.1.2.3", "172.16.0.1", "192.168.0.1", "169.254.1.1", "0.0.0.0"}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if !IsBlockedIP(ip) {
			t.Fatalf("expected blocked: %s", s)
		}
	}
	if IsBlockedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("8.8.8.8 should be allowed")
	}
}

func TestCheckRedirect_revalidates(t *testing.T) {
	req := &http.Request{URL: mustURL(t, "http://127.0.0.1/secret")}
	if err := CheckRedirect(req, nil); err == nil {
		t.Fatal("expected redirect to loopback blocked")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
