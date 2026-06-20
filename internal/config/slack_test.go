package config

import (
	"os"
	"testing"
)

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

func TestSlackDisabledByEnv(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_SLACK_DISABLED", "1")
	t.Setenv("NEURAL_JUNKIE_SLACK_ENABLED", "")
	if !SlackDisabledByEnv() {
		t.Fatal("expected disabled")
	}
	t.Setenv("NEURAL_JUNKIE_SLACK_DISABLED", "")
	t.Setenv("NEURAL_JUNKIE_SLACK_ENABLED", "0")
	if !SlackDisabledByEnv() {
		t.Fatal("expected disabled via ENABLED=0")
	}
}

func TestSlackConfigSlackReady(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_SLACK_DISABLED", "")
	t.Setenv("NEURAL_JUNKIE_SLACK_ENABLED", "")

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

	t.Setenv("NEURAL_JUNKIE_SLACK_DISABLED", "1")
	if ready.SlackReady() {
		t.Fatal("expected env disable to block ready config")
	}
	t.Setenv("NEURAL_JUNKIE_SLACK_DISABLED", "")
}

func TestMergeEnvVarsDisablesSlack(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_SLACK_DISABLED", "1")
	cfg := DefaultConfig()
	cfg.Slack.Enabled = true
	cfg.Slack.AppToken = "xapp-test"
	cfg.Slack.BotToken = "xoxb-test"
	cfg.mergeEnvVars()
	if cfg.Slack.Enabled {
		t.Fatal("mergeEnvVars should force Slack.Enabled false when disabled")
	}
	if cfg.Slack.SlackReady() {
		t.Fatal("SlackReady should be false when disabled")
	}
	os.Unsetenv("NEURAL_JUNKIE_SLACK_DISABLED")
}
