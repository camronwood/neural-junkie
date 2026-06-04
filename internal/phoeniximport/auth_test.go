package phoeniximport

import "testing"

func TestParseAuthConfigText(t *testing.T) {
	raw := `auth0 app:

Domain
dev-zazkmky7c1v5de5q.us.auth0.com

Client ID
abc123client

Client Secret
secret456
`
	cfg := parseAuthConfigText(raw)
	if cfg.Domain != "dev-zazkmky7c1v5de5q.us.auth0.com" {
		t.Fatalf("domain: %q", cfg.Domain)
	}
	if cfg.ClientID != "abc123client" {
		t.Fatalf("client id: %q", cfg.ClientID)
	}
	if cfg.ClientSecret != "secret456" {
		t.Fatalf("client secret: %q", cfg.ClientSecret)
	}
}

func TestDecodeJWTClaims(t *testing.T) {
	// payload: {"email":"u@example.com","brightestbio.com/email":"bb@x.com","brightestbio.com/organization":"Acme"}
	payload := "eyJlbWFpbCI6InVAZXhhbXBsZS5jb20iLCJicmlnaHRlc3RiaW8uY29tL2VtYWlsIjoiYmJAeC5jb20iLCJicmlnaHRlc3RiaW8uY29tL29yZ2FuaXphdGlvbiI6IkFjbWUifQ"
	token := "aaa." + payload + ".bbb"
	claims, err := decodeJWTClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.BrightestBioEmail != "bb@x.com" {
		t.Fatalf("email: %+v", claims)
	}
	id := identityFromToken(token)
	if id == "" || !contains(id, "bb@x.com") {
		t.Fatalf("identity: %q", id)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
