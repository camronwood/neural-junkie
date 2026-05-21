package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	key, err := LoadOrCreateSecretsKey()
	if err != nil {
		t.Fatal(err)
	}
	plain := "xoxb-secret-token"
	enc, err := encryptString(plain, key)
	if err != nil {
		t.Fatal(err)
	}
	if enc == plain || !strings.HasPrefix(enc, encryptedPrefix) {
		t.Fatalf("expected encrypted blob, got %q", enc)
	}
	dec, err := decryptString(enc, key)
	if err != nil || dec != plain {
		t.Fatalf("decrypt: %q err=%v", dec, err)
	}
}

func TestConfigSaveEncryptsSlackToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := DefaultConfig()
	cfg.Slack.BotToken = "xoxb-test"
	fp := filepath.Join(home, ".neural-junkie", "config.json")
	cfg.filePath = fp
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(fp)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "xoxb-test") {
		t.Fatal("plaintext token should not appear in config.json")
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Slack.BotToken != "xoxb-test" {
		t.Fatalf("loaded token %q", loaded.Slack.BotToken)
	}
}
