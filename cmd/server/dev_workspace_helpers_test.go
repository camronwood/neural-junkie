package main

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/filechange"
)

func TestLatestPendingChangeIDFiltersByChannel(t *testing.T) {
	t0 := time.Now().Add(-time.Minute)
	t1 := time.Now()
	pending := []*filechange.FileChange{
		{ID: "old-default", Channel: "default", RequestedAt: t0},
		{ID: "fast-a", Channel: "dev-fast-edit", RequestedAt: t0},
		{ID: "fast-b", Channel: "dev-fast-edit", RequestedAt: t1},
		{ID: "editor-x", Channel: "editor-ws1", RequestedAt: t1.Add(time.Second)},
	}
	if got := latestPendingChangeID(pending, "dev-fast-edit"); got != "fast-b" {
		t.Fatalf("dev-fast-edit: got %q want fast-b", got)
	}
	if got := latestPendingChangeID(pending, "editor-ws1"); got != "editor-x" {
		t.Fatalf("editor-ws1: got %q want editor-x", got)
	}
	if got := latestPendingChangeID(pending, "missing"); got != "" {
		t.Fatalf("missing channel: got %q want empty", got)
	}
}
