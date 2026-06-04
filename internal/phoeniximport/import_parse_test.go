package phoeniximport

import (
	"encoding/json"
	"testing"
)

func TestParseAttachmentNames(t *testing.T) {
	names := parseAttachmentNames([]byte(`["results.json","summary.zip"]`))
	if len(names) != 2 || names[0] != "results.json" {
		t.Fatalf("got %v", names)
	}
	names2 := parseAttachmentNames([]byte(`[{"name":"results"},{"filename":"summary.zip"}]`))
	if len(names2) != 2 {
		t.Fatalf("got %v", names2)
	}
}

func TestExtractScanResultsID(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"scanResultsId": "abc123",
		"nested":        map[string]any{"scanResultId": "ignored"},
	})
	if got := extractScanResultsID(raw); got != "abc123" {
		t.Fatalf("got %q", got)
	}
}

func TestPickScanZipAttachment(t *testing.T) {
	if got := pickScanZipAttachment([]string{"meta.json", "results"}); got != "results" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeDirName(t *testing.T) {
	if got := sanitizeDirName("507f1f77bcf86cd799439011"); got != "507f1f77bcf86cd799439011" {
		t.Fatalf("got %q", got)
	}
}
