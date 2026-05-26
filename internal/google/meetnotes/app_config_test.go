package meetnotes

import (
	"os"
	"path/filepath"
	"testing"
)

func clearGoogleOAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_ID", "")
	t.Setenv("NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_SECRET", "")
	t.Setenv("NEURAL_JUNKIE_GOOGLE_OAUTH_REDIRECT_URL", "")
}

func setVendorOAuthJSONForTest(t *testing.T, data []byte) {
	t.Helper()
	oldVendorOAuthJSON := vendorOAuthJSON
	t.Cleanup(func() {
		vendorOAuthJSON = oldVendorOAuthJSON
	})
	vendorOAuthJSON = data
}

func TestSaveAndResolveAppCredentials(t *testing.T) {
	clearGoogleOAuthEnv(t)
	setVendorOAuthJSONForTest(t, nil)
	dir := t.TempDir()
	creds := &AppOAuthCredentials{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:18765/api/assistant/google/callback",
	}
	if err := SaveAppCredentials(dir, creds); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveAppCredentials(dir)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ClientID != creds.ClientID || resolved.ClientSecret != creds.ClientSecret {
		t.Fatalf("resolved mismatch: %+v", resolved)
	}
	if !OAuthConfigured(dir) {
		t.Fatal("expected OAuthConfigured true")
	}
	path := filepath.Join(dir, appConfigFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("expected restrictive file mode, got %o", info.Mode().Perm())
	}
	pub := PublicAppConfigFromDir(dir)
	if !pub.ConnectReady || pub.OAuthSource != string(OAuthSourceConfig) {
		t.Fatalf("unexpected public config: %+v", pub)
	}
}

func TestResolveAppCredentialsFromEnv(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_ID", "env-client")
	t.Setenv("NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_SECRET", "env-secret")
	t.Setenv("NEURAL_JUNKIE_GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:18765/custom")

	resolved, src, err := ResolveAppCredentialsWithSource(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if src != OAuthSourceEnv {
		t.Fatalf("source = %q", src)
	}
	if resolved.ClientID != "env-client" || resolved.ClientSecret != "env-secret" || resolved.RedirectURL != "http://localhost:18765/custom" {
		t.Fatalf("resolved mismatch: %+v", resolved)
	}
}

func TestResolveAppCredentialsFromVendor(t *testing.T) {
	clearGoogleOAuthEnv(t)
	setVendorOAuthJSONForTest(t, []byte(`{
		"client_id": "vendor-client.apps.googleusercontent.com",
		"client_secret": "vendor-secret"
	}`))

	resolved, src, err := ResolveAppCredentialsWithSource(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if src != OAuthSourceVendor {
		t.Fatalf("source = %q", src)
	}
	if resolved.ClientID != "vendor-client.apps.googleusercontent.com" || resolved.ClientSecret != "vendor-secret" {
		t.Fatalf("resolved mismatch: %+v", resolved)
	}
	if resolved.RedirectURL != defaultRedirectURL {
		t.Fatalf("redirect = %q", resolved.RedirectURL)
	}
}

func TestPublicAppConfigUnconfigured(t *testing.T) {
	clearGoogleOAuthEnv(t)
	setVendorOAuthJSONForTest(t, nil)
	pub := PublicAppConfigFromDir(t.TempDir())
	if pub.ConnectReady || pub.Configured || pub.OAuthSource != string(OAuthSourceNone) {
		t.Fatalf("expected unavailable config, got %+v", pub)
	}
}
