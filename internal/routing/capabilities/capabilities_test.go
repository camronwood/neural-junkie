package capabilities

import (
	"testing"
)

func TestSelectOllamaTagFirstInstalled(t *testing.T) {
	p := &Profiles{
		TaskClasses: map[string][]string{
			"implement": {"qwen2.5-coder:14b", "deepseek-coder:6.7b"},
		},
	}
	installed := map[string]struct{}{"deepseek-coder:6.7b": {}}
	got := SelectOllamaTag(p, TaskImplement, installed, "qwen3.5:9b")
	if got.Tag != "deepseek-coder:6.7b" {
		t.Fatalf("tag = %q, want deepseek-coder:6.7b", got.Tag)
	}
	if got.Reason != "capability_implement" {
		t.Fatalf("reason = %q", got.Reason)
	}
}

func TestShouldUpgrade(t *testing.T) {
	p := &Profiles{
		TaskClasses: map[string][]string{
			"implement": {"qwen2.5-coder:14b", "qwen3.5:9b"},
		},
	}
	if !ShouldUpgrade(p, TaskImplement, "qwen3.5:9b", "qwen2.5-coder:14b") {
		t.Fatal("expected upgrade from qwen3.5 to qwen2.5-coder")
	}
	if ShouldUpgrade(p, TaskImplement, "qwen2.5-coder:14b", "qwen3.5:9b") {
		t.Fatal("should not downgrade")
	}
}

func TestLoadEmbedded(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.SourceRunID == "" {
		t.Fatal("expected source_run_id")
	}
	if len(p.Tags(TaskImplement)) == 0 {
		t.Fatal("expected implement tags")
	}
}

func TestClassifyCollabTask(t *testing.T) {
	if ClassifyCollabTask("Identify schema files in the repo") != TaskCollabLight {
		t.Fatal("want collab_light")
	}
	if ClassifyCollabTask("Write collabs/abc/findings.md with summary") != TaskImplement {
		t.Fatal("want implement for deliverable")
	}
}
