package collaboration

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/artifacts"
)

func TestSharedArtifactToCanvas(t *testing.T) {
	canvas, err := SharedArtifactToCanvas(&SharedArtifact{
		ID:      "plan-1",
		Title:   "Release plan",
		Content: "# Plan",
		Version: 2,
	}, artifacts.ArtifactLinks{CollaborationID: "collab-1"})
	if err != nil {
		t.Fatal(err)
	}
	if canvas == nil || canvas.Renderer.ID != "nj.markdown" || canvas.Revision != 2 {
		t.Fatalf("canvas=%+v", canvas)
	}
	if canvas.Links.CollaborationID != "collab-1" {
		t.Fatalf("links=%+v", canvas.Links)
	}
}
