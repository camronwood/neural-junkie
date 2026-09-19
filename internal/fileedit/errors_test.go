package fileedit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPatchError_structuredJSONFields(t *testing.T) {
	t.Parallel()
	err := EnrichWithFileHints(
		&PatchError{Code: ErrNotFound, Message: "old_string not found in file"},
		"package main\n\nfunc hello() {}\n",
		"func helo() {}",
	)
	pe, ok := err.(*PatchError)
	if !ok {
		t.Fatalf("expected PatchError, got %T", err)
	}
	if pe.Fingerprint == "" {
		t.Fatal("expected fingerprint")
	}
	if pe.Near == "" {
		t.Fatal("expected near hint")
	}
	if pe.Hint == "" {
		t.Fatal("expected hint")
	}
	raw := pe.JSONString()
	var payload map[string]string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("JSONString not valid JSON: %v (%s)", err, raw)
	}
	if payload["code"] != ErrNotFound {
		t.Fatalf("code=%q", payload["code"])
	}
	if payload["fingerprint"] == "" || payload["near"] == "" || payload["hint"] == "" {
		t.Fatalf("missing structured fields: %+v", payload)
	}
	if !strings.Contains(pe.Error(), `"code"`) {
		t.Fatalf("Error() should return structured JSON when enriched, got %q", pe.Error())
	}
}

func TestEnrichWithFileHints_notUniqueAndApplyFailed(t *testing.T) {
	t.Parallel()
	unique := EnrichWithFileHints(
		&PatchError{Code: ErrNotUnique, Message: "old_string matches 2 times"},
		"aaa\n",
		"a",
	).(*PatchError)
	if unique.Hint == "" || !strings.Contains(unique.Hint, "replace_all") {
		t.Fatalf("unique hint=%q", unique.Hint)
	}

	apply := EnrichWithFileHints(
		&PatchError{Code: ErrApplyFailed, Message: `context mismatch at line 2: expected "beta" got "BETA"`},
		"alpha\nBETA\ngamma\n",
		"beta",
	).(*PatchError)
	if apply.Fingerprint == "" {
		t.Fatal("expected fingerprint for apply_failed")
	}
	if apply.Hint == "" {
		t.Fatal("expected apply hint")
	}
}
