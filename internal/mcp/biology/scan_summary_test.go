package biology

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeScanSummaryPath(t *testing.T) {
	dir := t.TempDir()
	meta := filepath.Join(dir, "imageMetadata.json")
	payload := `{"metadata":[
		{"imageName":"A1","spots":[{"analyte":"BLANK","row":"1","column":"1","x_px":1,"y_px":1}]},
		{"imageName":"A2","spots":[]}
	]}`
	if err := os.WriteFile(meta, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := summarizeScanSummaryPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"Scan summary", "BLANK", "A2"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
	out2, err := summarizeScanSummaryPath(meta)
	if err != nil {
		t.Fatal(err)
	}
	if out2 == "" {
		t.Fatal("expected summary from metadata file path")
	}
}
