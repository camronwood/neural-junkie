package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsResponseFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{name: "bullet list", content: "Give me 5 tips as a bullet list", want: true},
		{name: "as bullets", content: "Reply as bullets please", want: true},
		{name: "numbered list", content: "as a numbered list of steps", want: true},
		{name: "numbered steps", content: "Give numbered steps to deploy", want: true},
		{name: "markdown table", content: "Reply as a markdown table with columns A|B", want: true},
		{name: "checklist", content: "Make a checklist for launch", want: true},
		{name: "as markdown", content: "Answer as markdown with headings", want: true},
		{name: "plain question", content: "What is TLS?", want: false},
		{name: "empty", content: "", want: false},
		{name: "bullet alone in prose", content: "The silver bullet approach failed", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := userRequestsResponseFormat(tc.content)
			if got != tc.want {
				t.Fatalf("userRequestsResponseFormat(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestGetResponseLengthGuidance_FormatRequests(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		content    string
		impl       bool
		wantSubstr string
		wantNot    string
	}{
		{
			name:       "bullet list",
			content:    "Give me 5 tips as a bullet list",
			wantSubstr: "unordered list",
			wantNot:    "2-5 sentences",
		},
		{
			name:       "numbered list",
			content:    "as a numbered list",
			wantSubstr: "ordered list",
			wantNot:    "2-5 sentences",
		},
		{
			name:       "markdown table",
			content:    "Reply as a markdown table with columns Name|Status",
			wantSubstr: "pipe table",
			wantNot:    "2-5 sentences",
		},
		{
			name:       "checklist",
			content:    "Give me a checklist for the release",
			wantSubstr: "checklist",
			wantNot:    "2-5 sentences",
		},
		{
			name:       "plain question stays concise",
			content:    "What is TLS?",
			wantSubstr: "2-5 sentences",
			wantNot:    "Honor it with real GitHub-flavored markdown",
		},
		{
			name:       "format wins over brief",
			content:    "brief bullet list of options",
			wantSubstr: "unordered list",
			wantNot:    "2-3 sentences max",
		},
		{
			name:       "implementation still wins over format",
			content:    "implement the fix and give me a bullet list of changes",
			impl:       true,
			wantSubstr: "[FILE_CHANGE]",
			wantNot:    "unordered list",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := getResponseLengthGuidance(tc.content, tc.impl)
			if !strings.Contains(got, tc.wantSubstr) {
				t.Fatalf("guidance missing %q:\n%s", tc.wantSubstr, got)
			}
			if tc.wantNot != "" && strings.Contains(got, tc.wantNot) {
				t.Fatalf("guidance unexpectedly contains %q:\n%s", tc.wantNot, got)
			}
		})
	}
}

func TestWritePersonaOpening_IncludesMarkdownStructure(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{
			Name:      "Assistant",
			Type:      protocol.AgentTypeAssistant,
			Expertise: []string{"general help"},
		},
	}
	msg := &protocol.Message{Channel: "dm-user"}

	var direct strings.Builder
	a.writePersonaOpening(&direct, msg, PersonaDirect)
	if !strings.Contains(direct.String(), "Respond naturally and conversationally") {
		t.Fatalf("direct persona missing conversational opener:\n%s", direct.String())
	}
	if !strings.Contains(direct.String(), "one list item per line") {
		t.Fatalf("direct persona missing markdown structure guidance:\n%s", direct.String())
	}

	var channel strings.Builder
	a.writePersonaOpening(&channel, msg, PersonaChannel)
	if !strings.Contains(channel.String(), "one list item per line") {
		t.Fatalf("channel persona missing markdown structure guidance:\n%s", channel.String())
	}

	var collab strings.Builder
	a.writePersonaOpening(&collab, msg, PersonaCollaboration)
	if strings.Contains(collab.String(), "one list item per line") {
		t.Fatalf("collaboration persona should not force structure guidance:\n%s", collab.String())
	}
}
