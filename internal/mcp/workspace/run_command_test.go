package workspace

import "testing"

func TestCommandAllowed_cargoBuildAndCheck(t *testing.T) {
	t.Parallel()
	for _, cmd := range []string{"cargo build", "cargo check", "cargo run", "cargo test"} {
		if !CommandAllowed(cmd) {
			t.Fatalf("expected %q allowlisted", cmd)
		}
	}
}
