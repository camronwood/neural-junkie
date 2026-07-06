package config

import "testing"

func TestAWSAccountAllowed(t *testing.T) {
	open := AWSConfig{}
	if !open.AccountAllowed("123456789012") {
		t.Fatal("empty allowlist should allow any account")
	}
	restricted := AWSConfig{AllowedAccounts: []string{"111111111111"}}
	if restricted.AccountAllowed("222222222222") {
		t.Fatal("expected account blocked")
	}
	if !restricted.AccountAllowed("111111111111") {
		t.Fatal("expected account allowed")
	}
}

func TestAWSWriteEnabledFlag(t *testing.T) {
	disabled := AWSConfig{}
	if disabled.WriteEnabledFlag() {
		t.Fatal("write should default false")
	}
	enabled := AWSConfig{WriteEnabled: func() *bool { v := true; return &v }()}
	if !enabled.WriteEnabledFlag() {
		t.Fatal("write_enabled true expected")
	}
}
