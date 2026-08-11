package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCorrectionRenameTarget(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Correction: rename the component to DisplayPreferences, but keep Appearance.", "DisplayPreferences"},
		{"refer to the component as DisplayPreferences", "DisplayPreferences"},
		{"call it ThemeSettings", "ThemeSettings"},
		{"keep the Appearance placement", ""},
	}
	for _, tc := range cases {
		if got := correctionRenameTarget(tc.in); got != tc.want {
			t.Fatalf("correctionRenameTarget(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSupersededComponentName(t *testing.T) {
	envelope := protocol.TurnContextEnvelope{
		Goal: &protocol.TurnContextGoal{
			Text: "Design a theme settings flow. Keep the toggle in an Appearance section and call the component ThemeSettings.",
		},
	}
	if got := supersededComponentName(envelope, "DisplayPreferences"); got != "ThemeSettings" {
		t.Fatalf("got %q want ThemeSettings", got)
	}
	if got := supersededComponentName(envelope, "ThemeSettings"); got != "" {
		t.Fatalf("same name should not supersede: %q", got)
	}
}

func TestValidateActiveCorrectionsHonored(t *testing.T) {
	envelope := protocol.TurnContextEnvelope{
		Goal: &protocol.TurnContextGoal{
			Text: "call the component ThemeSettings",
		},
		Corrections: []protocol.TurnContextCorrection{
			{Instruction: "rename the component to DisplayPreferences"},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "u"},
		"Continue from that and summarize the final design after the correction.")

	if issues := validateActiveCorrectionsHonored(envelope, msg, "Final design uses DisplayPreferences under Appearance."); len(issues) != 0 {
		t.Fatalf("expected pass, got %v", issues)
	}
	if issues := validateActiveCorrectionsHonored(envelope, msg, "Final design uses ThemeSettings under Appearance."); len(issues) == 0 {
		t.Fatal("expected correction_ignored for stale ThemeSettings summary")
	}
	if issues := validateActiveCorrectionsHonored(envelope, msg, "Keep Appearance placement."); len(issues) == 0 {
		t.Fatal("expected correction_ignored when DisplayPreferences missing")
	}
}

func TestValidateActiveCorrectionsHonored_pinnedGoalToken(t *testing.T) {
	envelope := protocol.TurnContextEnvelope{
		Goal: &protocol.TurnContextGoal{
			PinnedText: "Ship ThemeSettings under Appearance.",
			Text:       "Ship ThemeSettings under Appearance.",
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "u"},
		"summarize the final design after the correction.")
	if issues := validateActiveCorrectionsHonored(envelope, msg, "Final design keeps ThemeSettings."); len(issues) != 0 {
		t.Fatalf("expected pass when pinned token present, got %v", issues)
	}
	if issues := validateActiveCorrectionsHonored(envelope, msg, "Final design is a generic settings page."); len(issues) == 0 {
		t.Fatal("expected correction_ignored when pinned ThemeSettings dropped")
	}
}

func TestAppendDurableConversationContext_renameBanner(t *testing.T) {
	envelope := protocol.TurnContextEnvelope{
		Goal: &protocol.TurnContextGoal{Text: "call the component ThemeSettings"},
		Corrections: []protocol.TurnContextCorrection{
			{Instruction: "rename the component to DisplayPreferences"},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "u"}, "summarize")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	out := appendDurableConversationContext("BASE", envelope, msg)
	for _, needle := range []string{
		"Corrected name to use: DisplayPreferences",
		"Superseded name (do not use): ThemeSettings",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected %q in banner: %q", needle, out)
		}
	}
}
