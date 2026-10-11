// Package ssrf provides shared outbound-URL guards against SSRF.
package ssrf

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// IsBlockedIP reports whether ip is loopback, private, link-local, multicast,
// unspecified, or the common cloud metadata address.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// Explicit cloud metadata / APIPA (covered by link-local for IPv4, keep clear).
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
		return true
	}
	return false
}

// CheckPublicURL rejects non-http(s) schemes, localhost-style hostnames, and
// any DNS resolution that lands on a blocked IP (DNS rebinding defense).
func CheckPublicURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("SSRF: host %q not allowed", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return fmt.Errorf("SSRF: private/loopback IP not allowed")
		}
		return nil
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		// Fail closed: unresolved hosts are not fetchable for SSRF safety.
		return fmt.Errorf("SSRF: DNS lookup failed for %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("SSRF: no addresses for host %q", host)
	}
	for _, addr := range addrs {
		if IsBlockedIP(addr) {
			return fmt.Errorf("SSRF: host %q resolves to blocked address %s", host, addr)
		}
	}
	return nil
}

// CheckRedirect re-validates each redirect target (for http.Client.CheckRedirect).
func CheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if req == nil || req.URL == nil {
		return fmt.Errorf("SSRF: missing redirect URL")
	}
	return CheckPublicURL(req.URL.String())
}
