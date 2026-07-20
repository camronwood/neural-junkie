package contextcompress

import (
	"strings"
	"testing"
)

func TestCompressToolResult_grepUnderCap(t *testing.T) {
	raw := "a.go:1:main\nb.go:2:init"
	r := CompressToolResult(NewStore(10, 60, ""), "grep", "ch1", "c1", raw, DefaultOptions())
	if r.Strategy != StrategyNone {
		t.Fatalf("strategy = %q, want none", r.Strategy)
	}
	if r.Text != raw {
		t.Fatalf("unexpected text %q", r.Text)
	}
}

func TestCompressToolResult_grepLargeWithRetrieve(t *testing.T) {
	var lines []string
	for i := 0; i < 500; i++ {
		lines = append(lines, strings.Repeat("x", 40)+".go:"+itoa(i)+":match")
	}
	raw := strings.Join(lines, "\n")
	store := NewStore(100, 60, "")
	opts := DefaultOptions()
	opts.MaxToolBytes = 2000
	r := CompressToolResult(store, "grep", "dm-backend", "call-1", raw, opts)
	if r.Strategy != StrategyGrepTopN {
		t.Fatalf("strategy = %q", r.Strategy)
	}
	if r.Ref == "" {
		t.Fatal("expected ref")
	}
	if len(r.Text) >= len(raw) {
		t.Fatalf("expected compression, got %d vs %d", len(r.Text), len(raw))
	}
	if !strings.Contains(r.Text, "nj_retrieve_context") {
		t.Fatal("expected retrieve hint")
	}
	got, err := Retrieve(store, r.Ref, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Fatal("retrieve round-trip mismatch")
	}
}

func TestCompressToolResult_readFilePreview(t *testing.T) {
	var b strings.Builder
	b.WriteString("package main\n\n")
	for i := 0; i < 200; i++ {
		b.WriteString("// filler line\n")
	}
	b.WriteString("func main() {}\n")
	raw := b.String()
	opts := DefaultOptions()
	opts.MaxToolBytes = 1500
	r := CompressToolResult(NewStore(10, 60, ""), "read_file", "ch", "c1", raw, opts)
	if r.Strategy != StrategyReadPreview {
		t.Fatalf("strategy = %q", r.Strategy)
	}
	if !strings.Contains(r.Text, "func main") {
		t.Fatal("expected signature or tail in preview")
	}
}

func TestRetrieve_queryFilter(t *testing.T) {
	store := NewStore(10, 60, "")
	raw := "alpha\nbeta auth\ncharlie\n"
	ref := store.Put("ch", "c1", "grep", []byte(raw))
	out, err := Retrieve(store, ref, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if out != "beta auth" {
		t.Fatalf("got %q", out)
	}
}

func TestValidateContextRef_rejectsPlaceholders(t *testing.T) {
	cases := []string{"", "ctx-abc123", "ctx-example", "ctx-deadbeef", "not-a-ref", "CTX-ABC123"}
	for _, ref := range cases {
		if err := ValidateContextRef(ref); err == nil {
			t.Fatalf("expected error for %q", ref)
		}
	}
	good := "ctx-0123456789ab"
	if err := ValidateContextRef(good); err != nil {
		t.Fatalf("expected ok for %q: %v", good, err)
	}
}

func TestRetrieve_placeholderRef(t *testing.T) {
	store := NewStore(10, 60, "")
	_, err := Retrieve(store, "ctx-abc123", "")
	if err == nil {
		t.Fatal("expected error for placeholder ref")
	}
	if !strings.Contains(err.Error(), "documentation example") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompressSection(t *testing.T) {
	raw := strings.Repeat("summary ", 5000)
	opts := DefaultOptions()
	r := CompressSection(NewStore(10, 60, ""), "session_summary", "ch", "s1", raw, 2000, opts)
	if r.Strategy != StrategySection {
		t.Fatalf("strategy = %q", r.Strategy)
	}
	if r.Ref == "" {
		t.Fatal("expected ref for section")
	}
}

func TestStore_eviction(t *testing.T) {
	s := NewStore(2, 60, "")
	r1 := s.Put("a", "1", "t", []byte("one"))
	r2 := s.Put("a", "2", "t", []byte("two"))
	r3 := s.Put("a", "3", "t", []byte("three"))
	if _, ok := s.Get(r1); ok {
		t.Fatal("r1 should be evicted")
	}
	if _, ok := s.Get(r2); !ok {
		t.Fatal("r2 should remain")
	}
	if _, ok := s.Get(r3); !ok {
		t.Fatal("r3 should remain")
	}
}
