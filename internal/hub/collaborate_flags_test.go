package hub

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

func TestParseCollaborateLeadFlags_None(t *testing.T) {
	parts := []string{"/collaborate", "@a", "@b", "do", "thing"}
	parsed, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if parsed.Discussion.MaxRounds != 0 || parsed.Discussion.MaxTotalMessages != 0 {
		t.Fatalf("expected zero cfg, got %+v", parsed.Discussion)
	}
	if parsed.AttachWorkspace {
		t.Fatal("expected AttachWorkspace false")
	}
	if len(tail) != 4 || tail[0] != "@a" {
		t.Fatalf("unexpected tail: %#v", tail)
	}
}

func TestParseCollaborateLeadFlags_Both(t *testing.T) {
	parts := []string{"/collaborate", "--rounds", "5", "--messages", "40", "@x", "@y", "goal"}
	parsed, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if parsed.Discussion.MaxRounds != 5 || parsed.Discussion.MaxTotalMessages != 40 {
		t.Fatalf("cfg: %+v", parsed.Discussion)
	}
	if len(tail) != 3 || tail[0] != "@x" {
		t.Fatalf("tail: %#v", tail)
	}
	n := collaboration.DiscussionConfig{MaxRounds: parsed.Discussion.MaxRounds, MaxTotalMessages: parsed.Discussion.MaxTotalMessages}.Normalized()
	if n.MaxRounds != 5 || n.MaxTotalMessages != 40 {
		t.Fatalf("normalized: %+v", n)
	}
}

func TestParseCollaborateLeadFlags_Workspace(t *testing.T) {
	parts := []string{"/collaborate", "--workspace", "--rounds", "2", "@a", "@b", "goal"}
	parsed, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if !parsed.AttachWorkspace {
		t.Fatal("expected AttachWorkspace")
	}
	if parsed.Discussion.MaxRounds != 2 {
		t.Fatalf("rounds: %d", parsed.Discussion.MaxRounds)
	}
	if len(tail) != 3 {
		t.Fatalf("tail: %#v", tail)
	}
}

func TestParseCollaborateLeadFlags_MaxMessagesAlias(t *testing.T) {
	parts := []string{"/collaborate", "--max-messages", "12", "@a", "@b", "z"}
	parsed, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if parsed.Discussion.MaxTotalMessages != 12 || len(tail) != 3 {
		t.Fatalf("cfg %+v tail %#v", parsed.Discussion, tail)
	}
}

func TestParseCollaborateLeadFlags_MissingValue(t *testing.T) {
	parts := []string{"/collaborate", "--rounds"}
	_, _, err := parseCollaborateLeadFlags(parts)
	if err == "" {
		t.Fatal("expected error")
	}
}

func TestParseCollaborateLeadFlags_InvalidRoundsNumber(t *testing.T) {
	parts := []string{"/collaborate", "--rounds", "nope", "@a", "@b", "desc"}
	_, _, err := parseCollaborateLeadFlags(parts)
	if err == "" || !strings.Contains(err, "Invalid") {
		t.Fatalf("expected invalid rounds error, got %q", err)
	}
}

func TestParseCollaborateLeadFlags_InvalidMessagesNumber(t *testing.T) {
	parts := []string{"/collaborate", "--messages", "x", "@a", "@b", "desc"}
	_, _, err := parseCollaborateLeadFlags(parts)
	if err == "" || !strings.Contains(err, "Invalid") {
		t.Fatalf("expected invalid messages error, got %q", err)
	}
}

func TestParseCollaborateLeadFlags_Unknown(t *testing.T) {
	parts := []string{"/collaborate", "--typo", "1", "@a", "@b", "x"}
	_, _, err := parseCollaborateLeadFlags(parts)
	if err == "" {
		t.Fatal("expected error")
	}
}

func TestParseCollabExtendArgs_Both(t *testing.T) {
	parts := []string{"/collab-extend", "7c10c335", "--rounds", "2", "--messages", "6"}
	id, r, m, err := parseCollabExtendArgs(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if id != "7c10c335" || r != 2 || m != 6 {
		t.Fatalf("got id=%q rounds=%d msgs=%d", id, r, m)
	}
}

func TestParseCollabExtendArgs_PositionalRounds(t *testing.T) {
	parts := []string{"/collab-extend", "ec2cdef8", "1"}
	id, r, m, err := parseCollabExtendArgs(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if id != "ec2cdef8" || r != 1 || m != 0 {
		t.Fatalf("got id=%q rounds=%d msgs=%d", id, r, m)
	}
}

func TestParseCollabExtendArgs_PositionalBoth(t *testing.T) {
	parts := []string{"/collab-extend", "ec2cdef8", "2", "4"}
	id, r, m, err := parseCollabExtendArgs(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if id != "ec2cdef8" || r != 2 || m != 4 {
		t.Fatalf("got id=%q rounds=%d msgs=%d", id, r, m)
	}
}

func TestParseCollabExtendArgs_Errors(t *testing.T) {
	if _, _, _, err := parseCollabExtendArgs([]string{"/collab-extend"}); err == "" {
		t.Fatal("expected usage error")
	}
	if _, _, _, err := parseCollabExtendArgs([]string{"/collab-extend", "x", "junk"}); err == "" {
		t.Fatal("expected error for junk token")
	}
}
