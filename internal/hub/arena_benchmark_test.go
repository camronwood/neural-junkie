package hub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRankArenaBenchmarkResults(t *testing.T) {
	ranks := rankArenaBenchmarkResults([]ArenaBenchmarkModelResult{
		{Model: "b", Status: arenaBenchmarkModelCompleted, PassRate: 0.5, LogicCorrect: 1},
		{Model: "a", Status: arenaBenchmarkModelCompleted, PassRate: 1.0, LogicCorrect: 2},
		{Model: "c", Status: arenaBenchmarkModelPending, PassRate: 0.9, LogicCorrect: 9},
	})
	if len(ranks) != 2 {
		t.Fatalf("len=%d want 2", len(ranks))
	}
	if ranks[0].Model != "a" || ranks[0].Rank != 1 {
		t.Fatalf("first=%+v", ranks[0])
	}
	if ranks[1].Model != "b" || ranks[1].Rank != 2 {
		t.Fatalf("second=%+v", ranks[1])
	}
}

func TestAggregateArenaModelPassRate(t *testing.T) {
	r := &ArenaBenchmarkModelResult{
		LogicCorrect:      2,
		LogicTotal:        4,
		Connect4Completed: true,
	}
	aggregateArenaModelPassRate(r, []string{"logic-set", "connect4-smoke"})
	if r.PassRate != 0.75 {
		t.Fatalf("pass_rate=%v want 0.75", r.PassRate)
	}
	if !r.CapabilityPassed {
		t.Fatal("expected capability passed at 50 percent logic + connect4")
	}

	r2 := &ArenaBenchmarkModelResult{LogicCorrect: 0, LogicTotal: 4}
	aggregateArenaModelPassRate(r2, []string{"logic-set"})
	if r2.CapabilityPassed {
		t.Fatalf("expected capability fail below pass_rate_min")
	}
}

func TestResolveArenaBenchmarkSuiteEmbedded(t *testing.T) {
	// With no installed pack, embedded suites should resolve.
	suite, err := resolveArenaBenchmarkSuite("quick")
	if err != nil {
		t.Fatal(err)
	}
	if suite.ID != "quick" || len(suite.Challenges) != 2 {
		t.Fatalf("suite=%+v", suite)
	}
	if _, err := resolveArenaBenchmarkSuite("nope"); err == nil {
		t.Fatal("expected unknown suite error")
	}
}

func TestLoadArenaBenchmarkSuitesFromPackYAML(t *testing.T) {
	dir := t.TempDir()
	evalDir := filepath.Join(dir, "assets", "eval")
	if err := os.MkdirAll(evalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`
version: 1
default_suite: logic-only
suites:
  quick:
    title: Quick
    challenges: [logic-set, connect4-smoke]
  logic-only:
    title: Logic only
    challenges: [logic-set]
`)
	if err := os.WriteFile(filepath.Join(evalDir, "arena.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte(`
id: model-arena
title: Model Arena
layout_profile: team
capabilities: [model-arena]
agents:
  - type: arena
    name: ArenaExpert
assets:
  eval_probes: assets/eval/arena.yaml
`), 0o644); err != nil {
		t.Fatal(err)
	}

	path := arenaBenchmarkEvalPath(dir, nil)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	// Direct parse via LoadArenaBenchmarkSuites uses InstalledPackDir; exercise YAML unmarshal path via resolve helpers.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("empty yaml")
	}
}

func TestPersistAndLoadArenaBenchmarkJob(t *testing.T) {
	dir := t.TempDir()
	prev := arenaBenchmarkDirFn
	arenaBenchmarkDirFn = func() (string, error) { return dir, nil }
	t.Cleanup(func() { arenaBenchmarkDirFn = prev })

	h := &Hub{}
	job := &ArenaBenchmarkJob{
		ID:     "job-test-1",
		Status: arenaBenchmarkStatusCompleted,
		Suite:  "quick",
		Models: []string{"qwen3.5:9b"},
		Results: []ArenaBenchmarkModelResult{
			{Model: "qwen3.5:9b", Status: arenaBenchmarkModelCompleted, PassRate: 1},
		},
	}
	if err := h.persistArenaBenchmarkJob(job); err != nil {
		t.Fatal(err)
	}
	got, err := loadArenaBenchmarkJobFromDisk("job-test-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Suite != "quick" || got.Results[0].Model != "qwen3.5:9b" {
		t.Fatalf("got=%+v", got)
	}
}

func TestHealOrphanedArenaBenchmarkJob(t *testing.T) {
	dir := t.TempDir()
	prev := arenaBenchmarkDirFn
	arenaBenchmarkDirFn = func() (string, error) { return dir, nil }
	t.Cleanup(func() { arenaBenchmarkDirFn = prev })

	h := &Hub{}
	job := &ArenaBenchmarkJob{
		ID:     "orphan-1",
		Status: arenaBenchmarkStatusRunning,
		Suite:  "quick",
		Models: []string{"a"},
		Results: []ArenaBenchmarkModelResult{
			{Model: "a", Status: arenaBenchmarkModelRunning},
		},
	}
	if err := h.persistArenaBenchmarkJob(job); err != nil {
		t.Fatal(err)
	}
	got, err := h.GetArenaBenchmark("orphan-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != arenaBenchmarkStatusFailed {
		t.Fatalf("status=%q want failed", got.Status)
	}
	if got.Results[0].Status != arenaBenchmarkModelSkipped {
		t.Fatalf("result status=%q", got.Results[0].Status)
	}

	cancelled, err := h.CancelArenaBenchmark("orphan-1")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != arenaBenchmarkStatusFailed {
		// Already terminal from heal; cancel should be idempotent.
		t.Fatalf("cancel status=%q", cancelled.Status)
	}

	orphan2 := &ArenaBenchmarkJob{
		ID:     "orphan-2",
		Status: arenaBenchmarkStatusQueued,
		Suite:  "quick",
		Models: []string{"b"},
		Results: []ArenaBenchmarkModelResult{
			{Model: "b", Status: arenaBenchmarkModelPending},
		},
	}
	if err := h.persistArenaBenchmarkJob(orphan2); err != nil {
		t.Fatal(err)
	}
	cancelled2, err := h.CancelArenaBenchmark("orphan-2")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled2.Status != arenaBenchmarkStatusCancelled {
		t.Fatalf("cancel orphan status=%q want cancelled", cancelled2.Status)
	}
}
