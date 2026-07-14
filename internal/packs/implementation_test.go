package packs

import "testing"

func TestParseBuiltinImplementation(t *testing.T) {
	cases := []struct {
		in  string
		typ string
		ok  bool
	}{
		{"builtin/music", "music", true},
		{"  Builtin/Music  ", "music", true},
		{"builtin/", "", false},
		{"builtin/music/extra", "", false},
		{"music", "", false},
		{"", "", false},
		{"sidecar/music", "", false},
	}
	for _, tc := range cases {
		got, ok := ParseBuiltinImplementation(tc.in)
		if ok != tc.ok || got != tc.typ {
			t.Fatalf("%q: got (%q, %v), want (%q, %v)", tc.in, got, ok, tc.typ, tc.ok)
		}
	}
}
