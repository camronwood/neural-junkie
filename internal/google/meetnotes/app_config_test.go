package meetnotes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndResolveAppCredentials(t *testing.T) {
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
}
