package agent

import "testing"

func TestParseRunCommandMCPResult(t *testing.T) {
	code, out, _ := parseRunCommandMCPResult("exit_code=1\nbuild failed\n")
	if code != 1 || out != "build failed" {
		t.Fatalf("got code=%d out=%q", code, out)
	}
	code, out, _ = parseRunCommandMCPResult("exit_code=0\nok")
	if code != 0 || out != "ok" {
		t.Fatalf("got code=%d out=%q", code, out)
	}
}

func TestParseRunCommandToolInput(t *testing.T) {
	cmd := parseRunCommandToolInput([]byte(`{"command":"npm run build"}`))
	if cmd != "npm run build" {
		t.Fatalf("got %q", cmd)
	}
}
