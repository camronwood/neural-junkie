package phoeniximport

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAttachmentNamesTimObjects(t *testing.T) {
	raw := []byte(`[{"filename":"results.json","size":2289},{"filename":"summary.zip","size":14052}]`)
	names := parseAttachmentNames(raw)
	if len(names) != 2 || names[0] != "results.json" || names[1] != "summary.zip" {
		t.Fatalf("got %v", names)
	}
}

func TestListDocumentsOptionsEncode(t *testing.T) {
	opts := listDocumentsOptions{
		limit: 50,
		sort:  "-createdOn",
		query: map[string]any{"status": "COMPLETE"},
	}
	q, err := json.Marshal(opts.query)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join([]string{"limit=50", "sort=-createdOn", "query=" + string(q)}, "&")
	if !strings.Contains(joined, `"status":"COMPLETE"`) {
		t.Fatalf("unexpected query encoding: %s", joined)
	}
}
