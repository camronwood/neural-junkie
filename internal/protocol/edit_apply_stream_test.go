package protocol

import (
	"strings"
	"testing"
)

func TestChunkEditApplyContent(t *testing.T) {
	t.Parallel()

	if got := ChunkEditApplyContent("", 64); got != nil {
		t.Fatalf("empty content: got %#v, want nil", got)
	}

	chunks := ChunkEditApplyContent("abcdefghij", 4)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	joined := strings.Join(chunks, "")
	if joined != "abcdefghij" {
		t.Fatalf("rejoined %q, want original", joined)
	}
	for i, c := range chunks {
		if len(c) > 4 && i < len(chunks)-1 {
			t.Fatalf("chunk %d too large: %q (%d bytes)", i, c, len(c))
		}
	}

	unicode := ChunkEditApplyContent("こんにちは世界", 8)
	if strings.Join(unicode, "") != "こんにちは世界" {
		t.Fatalf("unicode rejoin failed: %#v", unicode)
	}
}

func TestParseEditApplyStreamMeta(t *testing.T) {
	t.Parallel()

	meta := map[string]interface{}{
		MetaStreamKind:        StreamKindEditApply,
		MetaEditApplyChangeID: "chg-1",
		MetaEditApplyPath:     "src/a.ts",
		MetaEditApplyOffset:   float64(12),
	}
	parsed, ok := ParseEditApplyStreamMeta(meta)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if parsed.ChangeID != "chg-1" || parsed.Path != "src/a.ts" || parsed.Offset != 12 {
		t.Fatalf("unexpected parse: %+v", parsed)
	}

	if _, ok := ParseEditApplyStreamMeta(map[string]interface{}{
		MetaStreamKind:        "chat",
		MetaEditApplyChangeID: "x",
		MetaEditApplyPath:     "y",
	}); ok {
		t.Fatal("non-edit_apply kind should fail")
	}
	if IsEditApplyStreamMeta(map[string]interface{}{"tool_step": "start"}) {
		t.Fatal("tool_step should not be edit_apply")
	}
}

func TestApplyEditApplyStreamMeta(t *testing.T) {
	t.Parallel()
	dst := map[string]interface{}{}
	ApplyEditApplyStreamMeta(dst, EditApplyStreamMeta{
		ChangeID: "c1",
		Path:     "f.go",
		Offset:   3,
		Done:     true,
	})
	if !IsEditApplyStreamMeta(dst) {
		t.Fatal("expected edit_apply after Apply")
	}
	if dst[MetaEditApplyDone] != true {
		t.Fatal("expected done flag")
	}
}
