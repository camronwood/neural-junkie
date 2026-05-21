package config

import "testing"

func TestSlackConfigEffectiveDisplayName(t *testing.T) {
	empty := SlackConfig{}
	if empty.EffectiveDisplayName() != "Camron" {
		t.Fatal("default display name")
	}
	if (&SlackConfig{DisplayName: "Alex"}).EffectiveDisplayName() != "Alex" {
		t.Fatal("custom display name")
	}
	var nilCfg *SlackConfig
	if nilCfg.EffectiveDisplayName() != "Camron" {
		t.Fatal("nil config default")
	}
}

func TestSlackConfigEffectiveDefaultPolicy(t *testing.T) {
	empty := SlackConfig{}
	if empty.EffectiveDefaultPolicy() != SlackPolicyMentionOnly {
		t.Fatal("default policy")
	}
	if (&SlackConfig{DefaultPolicy: SlackPolicyAlways}).EffectiveDefaultPolicy() != SlackPolicyAlways {
		t.Fatal("custom policy")
	}
}

func TestSlackConfigSlackReady(t *testing.T) {
	ready := SlackConfig{Enabled: true, AppToken: "xapp", BotToken: "xoxb"}
	if !ready.SlackReady() {
		t.Fatal("expected ready")
	}
	noApp := SlackConfig{Enabled: true, BotToken: "xoxb"}
	if noApp.SlackReady() {
		t.Fatal("missing app token")
	}
	disabled := SlackConfig{Enabled: false, AppToken: "xapp", BotToken: "xoxb"}
	if disabled.SlackReady() {
		t.Fatal("disabled")
	}
}
