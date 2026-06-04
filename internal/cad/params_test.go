package cad

import (
	"strings"
	"testing"
)

func TestParseParams(t *testing.T) {
	src := `/* [Dimensions] */
width = 20; // mm [10:100:5]
height = 10;
name = "bracket";
`
	params := ParseParams(src)
	if len(params) < 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	if params[0].Name != "width" || params[0].Section != "Dimensions" {
		t.Fatalf("width: %+v", params[0])
	}
	if params[0].Min == nil || params[0].Max == nil {
		t.Fatal("expected min/max on width")
	}
	if params[2].Name != "name" || !strings.Contains(params[2].Value, "bracket") {
		t.Fatalf("name param: %+v", params[2])
	}
}

func TestFormatOpenSCADValue(t *testing.T) {
	if formatOpenSCADValue("10") != "10" {
		t.Fatal("numeric")
	}
	if formatOpenSCADValue("true") != "true" {
		t.Fatal("bool")
	}
	if formatOpenSCADValue("hello") != `"hello"` {
		t.Fatal("string")
	}
}
