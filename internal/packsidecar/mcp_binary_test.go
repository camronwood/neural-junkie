package packsidecar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectMCPSidecarStub(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "sd-mcp-server")
	content := "#!/usr/bin/env bash\n# Test fixture stub for sd-mcp-server\n# --health-port=\n"
	if err := os.WriteFile(stub, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	if err := rejectMCPSidecarStub(stub); err == nil {
		t.Fatal("expected stub rejection")
	}

	realish := filepath.Join(dir, "real-bin")
	big := make([]byte, 33*1024)
	big[0] = '#'
	big[1] = '!'
	if err := os.WriteFile(realish, big, 0755); err != nil {
		t.Fatal(err)
	}
	if err := rejectMCPSidecarStub(realish); err != nil {
		t.Fatalf("large file should pass size gate: %v", err)
	}
}
