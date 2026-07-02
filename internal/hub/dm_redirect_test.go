package hub

import "testing"

func TestDMRedirectSetAndTake(t *testing.T) {
	ch := &CommandHandler{}
	ch.setDMRedirect("dm-user-swiftexpert")
	if name, ok := ch.TakeDMRedirect(); !ok || name != "dm-user-swiftexpert" {
		t.Fatalf("TakeDMRedirect() = %q, %v; want dm-user-swiftexpert, true", name, ok)
	}
	if name, ok := ch.TakeDMRedirect(); ok || name != "" {
		t.Fatalf("second TakeDMRedirect() = %q, %v; want empty, false", name, ok)
	}
}
