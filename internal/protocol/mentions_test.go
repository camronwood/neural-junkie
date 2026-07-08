package protocol

import "testing"

func TestParseMentionsIgnoresSlackUserIDs(t *testing.T) {
	got := ParseMentions("<@U0B5MLY2N2E> you there?")
	if len(got) != 0 {
		t.Fatalf("expected no mentions, got %v", got)
	}
}

func TestParseMentionsKeepsAgentNames(t *testing.T) {
	got := ParseMentions("@Assistant can you help?")
	if len(got) != 1 || got[0] != "assistant" {
		t.Fatalf("expected assistant mention, got %v", got)
	}
}

func TestIsSlackMentionToken(t *testing.T) {
	if !IsSlackMentionToken("u0b5mly2n2e") {
		t.Fatal("expected slack user id token")
	}
	if IsSlackMentionToken("assistant") {
		t.Fatal("agent name should not be slack token")
	}
	for _, name := range []string{"securityexpert", "softwarearchitect", "backendengineer", "frontendengineer"} {
		if IsSlackMentionToken(name) {
			t.Fatalf("agent name %q should not be slack token", name)
		}
	}
}

func TestParseMentionsKeepsSpecialistAgentNames(t *testing.T) {
	got := ParseMentions("@SoftwareArchitect @SecurityExpert design the auth flow")
	if len(got) != 2 {
		t.Fatalf("expected 2 mentions, got %v", got)
	}
}

func TestParseMentionsIgnoresScopedPathSyntax(t *testing.T) {
	got := ParseMentions("Fix @file:core/sample/math.go and @folder:src/ only")
	if len(got) != 0 {
		t.Fatalf("expected no agent mentions, got %v", got)
	}
	got = ParseMentions("@BackendEngineer check @file:main.go")
	if len(got) != 1 || got[0] != "backendengineer" {
		t.Fatalf("expected backendengineer only, got %v", got)
	}
}

func TestIsCollabTemplateMentionToken(t *testing.T) {
	if !IsCollabTemplateMentionToken("agentname") {
		t.Fatal("expected agentname template token")
	}
	if !IsCollabTemplateMentionToken("agent") {
		t.Fatal("expected agent template token")
	}
	if IsCollabTemplateMentionToken("backendengineer") {
		t.Fatal("real agent name should not be template token")
	}
}

func TestParseMentionsKeepsAgentNameProtocolContract(t *testing.T) {
	got := ParseMentions("@AgentName hello")
	if len(got) != 1 || got[0] != "agentname" {
		t.Fatalf("expected agentname mention for protocol contract, got %v", got)
	}
}

func TestFilterCollabTemplateMentions(t *testing.T) {
	got := FilterCollabTemplateMentions([]string{"agentname", "backendengineer", "agent"})
	if len(got) != 1 || got[0] != "backendengineer" {
		t.Fatalf("expected only backendengineer, got %v", got)
	}
}

func TestSlackSenderSkipsMentionParsing(t *testing.T) {
	msg := NewMessage(
		MessageTypeQuestion,
		"slack:inbox:U1",
		AgentInfo{ID: "slack:U1", Name: "Camron", Type: AgentTypeGeneral},
		"<@U0B5MLY2N2E> what model are you running?",
	)
	if len(msg.Mentions) != 0 {
		t.Fatalf("expected no mentions for slack sender, got %v", msg.Mentions)
	}
}
