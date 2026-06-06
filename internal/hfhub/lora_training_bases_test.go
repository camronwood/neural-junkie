package hfhub

import (
	"strings"
	"testing"
)

func TestValidateLoRATrainingBaseRejectsQwen(t *testing.T) {
	err := ValidateLoRATrainingBase("qwen2.5-coder:14b")
	if err == nil {
		t.Fatal("expected error for qwen base")
	}
	if !strings.Contains(err.Error(), "Qwen") {
		t.Fatalf("expected Qwen in error: %v", err)
	}
}

func TestValidateLoRATrainingBaseAcceptsLlama31(t *testing.T) {
	if err := ValidateLoRATrainingBase("llama3.1:8b"); err != nil {
		t.Fatal(err)
	}
}

func TestMapLoRABaseToHF(t *testing.T) {
	if got := MapLoRABaseToHF("codestral:latest"); got != "mistralai/Codestral-22B-v0.1" {
		t.Fatalf("codestral hf = %q", got)
	}
	if got := MapLoRABaseToHF("llama3:8b"); got != "meta-llama/Meta-Llama-3-8B-Instruct" {
		t.Fatalf("llama3 hf = %q", got)
	}
}

func TestDefaultLoRATrainingBaseForAgent(t *testing.T) {
	if got := DefaultLoRATrainingBaseForAgent("biology"); got != BiologyLoRABaseTag {
		t.Fatalf("biology base = %q", got)
	}
	if got := DefaultLoRATrainingBaseForAgent("backend"); got != DefaultLoRATrainingCodeBase {
		t.Fatalf("code base = %q", got)
	}
}
