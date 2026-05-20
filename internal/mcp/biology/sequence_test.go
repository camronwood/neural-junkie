package biology

import (
	"strings"
	"testing"
)

func TestAnalyzeSequenceProtein(t *testing.T) {
	out, err := analyzeSequenceText("MKTAYIAKQRQISFVK")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "protein") {
		t.Fatalf("expected protein type in output: %s", out)
	}
}

func TestAnalyzeSequenceDNA(t *testing.T) {
	out, err := analyzeSequenceText(">gene1\nATGCGT\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "DNA") {
		t.Fatalf("expected DNA in output: %s", out)
	}
}
