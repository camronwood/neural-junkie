package slack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOAuthAppSaveLoadPublic(t *testing.T) {
	useTempHomeDir(t)
	if err := SaveOAuthApp(&OAuthAppCredentials{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/cb",
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadOAuthApp()
	if err != nil || loaded == nil {
		t.Fatalf("load: %v %+v", err, loaded)
	}
	if loaded.ClientSecret != "secret" {
		t.Fatal("secret mismatch")
	}
	pub := PublicOAuthFromDir()
	if !pub.Configured || !pub.SecretSet || pub.ClientID != "cid" {
		t.Fatalf("%+v", pub)
	}
	p, err := oauthAppPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "oauth_app.json" {
		t.Fatalf("path %q", p)
	}
}

func TestBaseDirCreatesDirectory(t *testing.T) {
	useTempHomeDir(t)
	dir, err := BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}
