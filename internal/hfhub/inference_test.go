package hfhub

import "testing"

func TestInferenceModelURL(t *testing.T) {
	got := InferenceModelURL("facebook/esmfold_v1")
	want := "https://router.huggingface.co/hf-inference/models/facebook/esmfold_v1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
