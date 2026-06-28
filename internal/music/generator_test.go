package music

import (
	"context"
	"testing"
)

func TestResolveGeneratorPrefersDefault(t *testing.T) {
	oldDefault := Default
	oldSidecar := SidecarBaseURL
	t.Cleanup(func() {
		Default = oldDefault
		SidecarBaseURL = oldSidecar
	})
	Default = stubGen{}
	SidecarBaseURL = func() string { return "http://example:9999" }
	if _, ok := ResolveGenerator().(stubGen); !ok {
		t.Fatal("expected Default generator")
	}
}

func TestResolveGeneratorLazySidecar(t *testing.T) {
	oldDefault := Default
	oldSidecar := SidecarBaseURL
	t.Cleanup(func() {
		Default = oldDefault
		SidecarBaseURL = oldSidecar
	})
	Default = nil
	SidecarBaseURL = func() string { return "http://127.0.0.1:8765" }
	gen := ResolveGenerator()
	sg, ok := gen.(*SidecarGenerator)
	if !ok || sg == nil {
		t.Fatalf("expected SidecarGenerator, got %T", gen)
	}
}

type stubGen struct{}

func (stubGen) Generate(context.Context, Request) (string, string, error) { return "", "", nil }
