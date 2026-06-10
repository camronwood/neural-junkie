package codereview

import "testing"

func TestIsValidGoPath(t *testing.T) {
	c := &CodeReviewMCP{}
	if !c.isValidGoPath(".") {
		t.Fatal("expected . allowed")
	}
	if c.isValidGoPath("../etc/passwd") {
		t.Fatal("expected traversal rejected")
	}
	if c.isValidGoPath("") {
		t.Fatal("expected empty rejected")
	}
}
