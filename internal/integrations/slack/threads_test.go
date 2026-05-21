package slack

import (
	"os"
	"testing"
)

func TestThreadMapResolveInboundTopLevel(t *testing.T) {
	tm := &ThreadMap{roots: map[string]map[string]string{}, njToSlack: map[string]string{}}
	tid, reply, isThread := tm.ResolveInbound("C1", "100.1", "")
	if isThread || tid != "" || reply != "" {
		t.Fatalf("top-level: tid=%q reply=%q isThread=%v", tid, reply, isThread)
	}
}

func TestThreadMapResolveInboundFirstReply(t *testing.T) {
	tm := &ThreadMap{roots: map[string]map[string]string{}, njToSlack: map[string]string{}}
	tid, reply, isThread := tm.ResolveInbound("C1", "100.2", "100.1")
	if !isThread || tid != "100.1" || reply != "100.1" {
		t.Fatalf("first reply: tid=%q reply=%q isThread=%v", tid, reply, isThread)
	}
}

func TestThreadMapResolveInboundKnownRoot(t *testing.T) {
	tm := &ThreadMap{
		roots: map[string]map[string]string{
			"C1": {"100.1": "nj-root-id"},
		},
		njToSlack: map[string]string{"nj-root-id": "100.1"},
	}
	tid, reply, isThread := tm.ResolveInbound("C1", "100.5", "100.1")
	if !isThread || tid != "nj-root-id" || reply != "100.5" {
		t.Fatalf("known root: tid=%q reply=%q isThread=%v", tid, reply, isThread)
	}
}

func TestThreadMapRegisterInboundAndOutbound(t *testing.T) {
	useTempHomeDir(t)
	tm, err := NewThreadMap()
	if err != nil {
		t.Fatal(err)
	}
	if err := tm.RegisterInboundRoot("C1", "200.1", "nj-root"); err != nil {
		t.Fatal(err)
	}
	if ts := tm.SlackThreadTS("nj-root"); ts != "200.1" {
		t.Fatalf("slack ts %q", ts)
	}
	if err := tm.RegisterOutbound("nj-root", "200.9"); err != nil {
		t.Fatal(err)
	}
	if ts := tm.SlackThreadTS("nj-root"); ts != "200.9" {
		t.Fatalf("updated slack ts %q", ts)
	}
	tm2, err := NewThreadMap()
	if err != nil {
		t.Fatal(err)
	}
	if ts := tm2.SlackThreadTS("nj-root"); ts != "200.9" {
		t.Fatalf("reloaded slack ts %q", ts)
	}
}

func TestThreadMapRegisterOutboundNoOp(t *testing.T) {
	tm := &ThreadMap{filePath: os.DevNull, njToSlack: map[string]string{}}
	if err := tm.RegisterOutbound("", "1"); err != nil {
		t.Fatal(err)
	}
}
