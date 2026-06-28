package music

import "testing"

func TestNormalizeModelVariant(t *testing.T) {
	if got := NormalizeModelVariant("xl-turbo"); got != "xl-turbo" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeModelVariant("unknown"); got != "sft" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestCheckpointNameForVariant(t *testing.T) {
	if got := CheckpointNameForVariant("xl-sft"); got != "acestep-v15-xl-sft" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultInferenceSteps(t *testing.T) {
	if DefaultInferenceSteps("sft") != 50 {
		t.Fatal("sft steps")
	}
	if DefaultInferenceSteps("turbo") != 8 {
		t.Fatal("turbo steps")
	}
}
