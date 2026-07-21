package protocol

import "testing"

func TestSetArtifactReference(t *testing.T) {
	msg := &Message{}
	ref := ArtifactReference{
		ID:                 "artifact-1",
		RendererID:         "nj.chart",
		RendererAPIVersion: 1,
		Revision:           2,
		Action:             "updated",
	}
	msg.SetArtifactReference(ref)
	got, ok := msg.Metadata["artifact_ref"].(ArtifactReference)
	if !ok {
		t.Fatalf("artifact_ref type = %T", msg.Metadata["artifact_ref"])
	}
	if got.ID != ref.ID || got.Revision != ref.Revision {
		t.Fatalf("artifact_ref = %+v", got)
	}
}
