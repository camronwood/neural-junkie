package ollama_test

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ollama"
)

func TestLibrary(t *testing.T) {
	models, err := ollama.Library()
	if err != nil {
		t.Fatal(err)
	}
	if len(models) == 0 {
		t.Fatal("expected non-empty embedded catalog")
	}
	if models[0].Name == "" || models[0].Title == "" {
		t.Fatalf("first entry missing name/title: %#v", models[0])
	}
	for _, m := range models {
		if m.Name == "" {
			t.Errorf("catalog row missing name: %#v", m)
		}
	}
}
