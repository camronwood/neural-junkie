package pathutil

import (
	"os"
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

func TestWithinRoot_rejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("classified"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	_, err := WithinRoot(root, filepath.Join(link, "secret.txt"))
	if err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
	if !strings.Contains(err.Error(), "outside") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestWithinRoot_allowsInsideSymlink(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "real")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(inner, "ok.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(inner, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	got, err := WithinRoot(root, filepath.Join(link, "ok.txt"))
	if err != nil {
		t.Fatalf("inside symlink should be allowed: %v", err)
	}
	// Returned under caller's root form; still names the real file via Rel.
	want := filepath.Join(root, "real", "ok.txt")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
