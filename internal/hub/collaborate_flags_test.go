package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

func TestParseCollaborateLeadFlags_None(t *testing.T) {
	parts := []string{"/collaborate", "@a", "@b", "do", "thing"}
	cfg, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if cfg.MaxRounds != 0 || cfg.MaxTotalMessages != 0 {
		t.Fatalf("expected zero cfg, got %+v", cfg)
	}
	if len(tail) != 4 || tail[0] != "@a" {
		t.Fatalf("unexpected tail: %#v", tail)
	}
}

func TestParseCollaborateLeadFlags_Both(t *testing.T) {
	parts := []string{"/collaborate", "--rounds", "5", "--messages", "40", "@x", "@y", "goal"}
	cfg, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if cfg.MaxRounds != 5 || cfg.MaxTotalMessages != 40 {
		t.Fatalf("cfg: %+v", cfg)
	}
	if len(tail) != 3 || tail[0] != "@x" {
		t.Fatalf("tail: %#v", tail)
	}
	n := collaboration.DiscussionConfig{MaxRounds: cfg.MaxRounds, MaxTotalMessages: cfg.MaxTotalMessages}.Normalized()
	if n.MaxRounds != 5 || n.MaxTotalMessages != 40 {
		t.Fatalf("normalized: %+v", n)
	}
}

func TestParseCollaborateLeadFlags_MaxMessagesAlias(t *testing.T) {
	parts := []string{"/collaborate", "--max-messages", "12", "@a", "@b", "z"}
	cfg, tail, err := parseCollaborateLeadFlags(parts)
	if err != "" {
		t.Fatalf("unexpected err: %s", err)
	}
	if cfg.MaxTotalMessages != 12 || len(tail) != 3 {
		t.Fatalf("cfg %+v tail %#v", cfg, tail)
	}
}

func TestParseCollaborateLeadFlags_MissingValue(t *testing.T) {
	parts := []string{"/collaborate", "--rounds"}
	_, _, err := parseCollaborateLeadFlags(parts)
	if err == "" {
		t.Fatal("expected error")
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

func TestParseCollabExtendArgs_Errors(t *testing.T) {
	if _, _, _, err := parseCollabExtendArgs([]string{"/collab-extend"}); err == "" {
		t.Fatal("expected usage error")
	}
	if _, _, _, err := parseCollabExtendArgs([]string{"/collab-extend", "x", "junk"}); err == "" {
		t.Fatal("expected error for junk token")
	}
}
