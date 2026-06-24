package hub

import "testing"

func TestUserMayAccessDMChannel_scenarioOwner(t *testing.T) {
	if !userMayAccessDMChannel("chatscenario", "dm-chatscenario-backendengineer", "ChatScenario") {
		t.Fatal("channel owner should access DM")
	}
}

func TestUserMayAccessDMChannel_apiKeyAutomation(t *testing.T) {
	if !userMayAccessDMChannel("apikey", "dm-chatscenario-backendengineer", "ChatScenario") {
		t.Fatal("automation API key should access scenario-owned DM")
	}
	if !userMayAccessDMChannel("apikey", "dm-deliverablejudge-gemini", "DeliverableJudge") {
		t.Fatal("automation API key should access judge DM")
	}
}

func TestUserMayAccessDMChannel_deniesForeignDM(t *testing.T) {
	if userMayAccessDMChannel("apikey", "dm-alice-backendengineer", "ChatScenario") {
		t.Fatal("automation should not access unrelated user DM")
	}
	if userMayAccessDMChannel("other", "dm-chatscenario-backendengineer", "ChatScenario") {
		t.Fatal("unrelated user should not access ChatScenario DM")
	}
}
