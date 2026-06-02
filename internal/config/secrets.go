package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const encryptedPrefix = "enc:v1:"

func secretsKeyPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "secrets.key"), nil
}

// LoadOrCreateSecretsKey returns the 32-byte AES key from NEURAL_JUNKIE_CONFIG_KEY (64 hex chars)
// or ~/.neural-junkie/secrets.key (created with mode 0600 on first use).
func LoadOrCreateSecretsKey() ([]byte, error) {
	if env := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_CONFIG_KEY")); env != "" {
		raw, err := hex.DecodeString(env)
		if err != nil || len(raw) != 32 {
			return nil, fmt.Errorf("NEURAL_JUNKIE_CONFIG_KEY must be 64 hex characters (32 bytes)")
		}
		return raw, nil
	}
	path, err := secretsKeyPath()
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(path); err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("secrets.key must be 32 bytes")
		}
		return data, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create secrets dir: %w", err)
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("create secrets.key: %w", err)
	}
	return key, nil
}

func encryptString(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptString(blob string, key []byte) (string, error) {
	if blob == "" {
		return "", nil
	}
	if !strings.HasPrefix(blob, encryptedPrefix) {
		return blob, nil // legacy plaintext
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(blob, encryptedPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deriveLegacyKey(home string) []byte {
	sum := sha256.Sum256([]byte("neural-junkie-config:" + home))
	return sum[:]
}

// EncryptSecret encrypts a string for storage outside config.json.
func EncryptSecret(plaintext string) (string, error) {
	key, err := LoadOrCreateSecretsKey()
	if err != nil {
		return "", err
	}
	return encryptString(plaintext, key)
}

// DecryptSecret decrypts a string stored with EncryptSecret.
func DecryptSecret(blob string) (string, error) {
	key, err := LoadOrCreateSecretsKey()
	if err != nil {
		return "", err
	}
	return decryptString(blob, key)
}

// EncryptConfigSecrets encrypts sensitive fields in-place before persisting config.json.
func (c *Config) EncryptSecretsBeforeSave() error {
	key, err := LoadOrCreateSecretsKey()
	if err != nil {
		return err
	}
	c.Slack.AppToken, err = encryptString(c.Slack.AppToken, key)
	if err != nil {
		return err
	}
	c.Slack.BotToken, err = encryptString(c.Slack.BotToken, key)
	if err != nil {
		return err
	}
	c.Slack.ClientSecret, err = encryptString(c.Slack.ClientSecret, key)
	if err != nil {
		return err
	}
	c.HF.Token, err = encryptString(c.HF.Token, key)
	if err != nil {
		return err
	}
	for i := range c.AI.Providers {
		c.AI.Providers[i].APIKey, err = encryptString(c.AI.Providers[i].APIKey, key)
		if err != nil {
			return err
		}
	}
	return nil
}

// DecryptConfigSecrets decrypts sensitive fields after loading config.json.
func (c *Config) DecryptSecretsAfterLoad() error {
	key, err := LoadOrCreateSecretsKey()
	if err != nil {
		// First run: no key file yet; try legacy home-derived key for migration
		home, herr := os.UserHomeDir()
		if herr != nil {
			return err
		}
		key = deriveLegacyKey(home)
	}
	var decErr error
	dec := func(s string) string {
		out, err := decryptString(s, key)
		if err != nil {
			decErr = err
			return s
		}
		return out
	}
	c.Slack.AppToken = dec(c.Slack.AppToken)
	c.Slack.BotToken = dec(c.Slack.BotToken)
	c.Slack.ClientSecret = dec(c.Slack.ClientSecret)
	c.HF.Token = dec(c.HF.Token)
	for i := range c.AI.Providers {
		c.AI.Providers[i].APIKey = dec(c.AI.Providers[i].APIKey)
	}
	return decErr
}
