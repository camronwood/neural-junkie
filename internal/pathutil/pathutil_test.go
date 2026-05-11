package pathutil

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertWithinRootAbs(t *testing.T) {
	root := filepath.FromSlash("/tmp/ws")
	child := filepath.FromSlash("/tmp/ws/a/b")
	outside := filepath.FromSlash("/tmp/ws_other/file")
	sibling := filepath.FromSlash("/tmp/ws2")

	if err := AssertWithinRootAbs(root, root); err != nil {
		t.Fatalf("same path: %v", err)
	}
	if err := AssertWithinRootAbs(root, child); err != nil {
		t.Fatalf("child: %v", err)
	}
	if err := AssertWithinRootAbs(root, outside); err == nil {
		t.Fatal("expected error for prefix sibling path")
	}
	if err := AssertWithinRootAbs(root, sibling); err == nil {
		t.Fatal("expected error for /tmp/ws2 vs /tmp/ws")
	}
	// Classic prefix bug: "/tmp/ws" must not contain "/tmp/ws_extra"
	bad := filepath.FromSlash("/tmp/ws_extra/oops")
	err := AssertWithinRootAbs(root, bad)
	if err == nil {
		t.Fatal("expected error for ws_extra")
	}
	if !strings.Contains(err.Error(), "outside") {
		t.Fatalf("wrong error: %v", err)
	}
}
