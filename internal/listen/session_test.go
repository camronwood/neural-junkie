package listen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type stubTranscriber struct {
	ready     bool
	hint      string
	text      string
	err       error
	callCount int
	mu        sync.Mutex
}

func (s *stubTranscriber) Ready() (bool, string) { return s.ready, s.hint }
func (s *stubTranscriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	s.mu.Lock()
	s.callCount++
	s.mu.Unlock()
	if s.err != nil {
		return "", s.err
	}
	return s.text, nil
}

func TestStartChunkStopHappyPath(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	var ingested struct {
		title, transcript string
	}
	m := NewManager(&stubTranscriber{ready: true, text: "hello world"}, func(ctx context.Context, title, transcript string) (string, string, string, error) {
		ingested.title = title
		ingested.transcript = transcript
		return "note-1", title, "summary", nil
	}, true, tmp)

	id, err := m.Start("Standup")
	if err != nil {
		t.Fatal(err)
	}
	partial, err := m.Chunk(context.Background(), id, []byte("RIFF....fake"))
	if err != nil {
		t.Fatal(err)
	}
	if partial != "hello world" {
		t.Fatalf("partial=%q", partial)
	}
	res, err := m.Stop(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if res.NoteID != "note-1" || ingested.transcript != "hello world" {
		t.Fatalf("result=%+v ingested=%+v", res, ingested)
	}
	if st := m.Status(); st.Active {
		t.Fatal("expected inactive after stop")
	}
}

func TestDoubleStartRejected(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: true, text: "x"}, nil, true, t.TempDir())
	if _, err := m.Start(""); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(""); !errors.Is(err, ErrAlreadyListening) {
		t.Fatalf("want ErrAlreadyListening, got %v", err)
	}
}

func TestStopNoSession(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: true}, nil, true, t.TempDir())
	if _, err := m.Stop(context.Background(), ""); !errors.Is(err, ErrNoSession) {
		t.Fatalf("got %v", err)
	}
}

func TestCancelMidListen(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	m := NewManager(&stubTranscriber{ready: true, text: "a"}, nil, true, tmp)
	id, err := m.Start("t")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Chunk(context.Background(), id, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := m.Cancel(id); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Stop(context.Background(), id); !errors.Is(err, ErrNoSession) {
		t.Fatalf("got %v", err)
	}
	// temp dir for session should be gone
	entries, _ := os.ReadDir(tmp)
	if len(entries) != 0 {
		t.Fatalf("expected cleaned tmp, got %v", entries)
	}
}

func TestSTTUnavailable(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: false, hint: "missing model"}, nil, true, t.TempDir())
	_, err := m.Start("")
	if !errors.Is(err, ErrSTTUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestChunkTooLarge(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: true, text: "ok"}, nil, true, t.TempDir())
	m.maxChunkBytes = 4
	id, err := m.Start("")
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Chunk(context.Background(), id, []byte("12345"))
	if !errors.Is(err, ErrChunkTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestEmptyChunkKeepsPartial(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: true, text: "hi"}, nil, true, t.TempDir())
	id, _ := m.Start("")
	_, _ = m.Chunk(context.Background(), id, []byte("x"))
	partial, err := m.Chunk(context.Background(), id, nil)
	if err != nil || partial != "hi" {
		t.Fatalf("partial=%q err=%v", partial, err)
	}
}

func TestStopEmptyTranscript(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: true, text: "   "}, nil, true, t.TempDir())
	id, _ := m.Start("")
	_, _ = m.Chunk(context.Background(), id, []byte("x"))
	_, err := m.Stop(context.Background(), id)
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("got %v", err)
	}
}

func TestSTTFailureSurfaces(t *testing.T) {
	t.Parallel()
	m := NewManager(&stubTranscriber{ready: true, err: errors.New("boom")}, nil, true, t.TempDir())
	id, _ := m.Start("")
	_, err := m.Chunk(context.Background(), id, []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("got %v", err)
	}
	st := m.Status()
	if st.Error == "" {
		t.Fatal("expected last error on status")
	}
}

func TestCancelDuringFinalize(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	m := NewManager(&stubTranscriber{ready: true, text: "done"}, func(ctx context.Context, title, transcript string) (string, string, string, error) {
		close(started)
		<-release
		return "n", title, "s", nil
	}, true, t.TempDir())
	id, _ := m.Start("")
	_, _ = m.Chunk(context.Background(), id, []byte("x"))

	errCh := make(chan error, 1)
	go func() {
		_, err := m.Stop(context.Background(), id)
		errCh <- err
	}()
	<-started
	// Cancel while finalizing: session still present until Stop clears it.
	// Concurrent cancel should race; either ErrNoSession after stop or cancel succeeds first.
	_ = m.Cancel(id)
	close(release)
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("stop hung")
	}
}

func TestChunkOrderPreservedUnderConcurrency(t *testing.T) {
	t.Parallel()
	d := &delayTranscriber{delay: 30 * time.Millisecond}
	m := NewManager(d, func(ctx context.Context, title, transcript string) (string, string, string, error) {
		return "n", title, transcript, nil
	}, true, t.TempDir())
	id, err := m.Start("t")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, label := range []string{"one", "two", "three"} {
		wg.Add(1)
		go func(label string) {
			defer wg.Done()
			_, err := m.Chunk(context.Background(), id, []byte(label))
			if err != nil {
				t.Errorf("chunk %s: %v", label, err)
			}
		}(label)
		time.Sleep(5 * time.Millisecond) // start order
	}
	wg.Wait()
	res, err := m.Stop(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if res.Transcript != "one two three" {
		t.Fatalf("transcript order=%q want %q", res.Transcript, "one two three")
	}
}

type delayTranscriber struct {
	delay time.Duration
}

func (d *delayTranscriber) Ready() (bool, string) { return true, "" }
func (d *delayTranscriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	data, err := os.ReadFile(audioPath)
	if err != nil {
		return "", err
	}
	time.Sleep(d.delay)
	return string(data), nil
}

func TestStopWaitsForInFlightChunk(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	tstub := &gateTranscriber{started: started, release: release}
	m := NewManager(tstub, func(ctx context.Context, title, transcript string) (string, string, string, error) {
		return "n", title, transcript, nil
	}, true, t.TempDir())
	id, _ := m.Start("t")
	errCh := make(chan error, 1)
	go func() {
		_, err := m.Chunk(context.Background(), id, []byte("late"))
		errCh <- err
	}()
	<-started
	stopCh := make(chan *StopResult, 1)
	go func() {
		res, err := m.Stop(context.Background(), id)
		if err != nil {
			t.Errorf("stop: %v", err)
			stopCh <- nil
			return
		}
		stopCh <- res
	}()
	time.Sleep(20 * time.Millisecond)
	close(release)
	<-errCh
	res := <-stopCh
	if res == nil || res.Transcript != "late" {
		t.Fatalf("expected stop to include in-flight chunk, got %#v", res)
	}
}

type gateTranscriber struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *gateTranscriber) Ready() (bool, string) { return true, "" }
func (g *gateTranscriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	g.once.Do(func() { close(g.started) })
	<-g.release
	data, err := os.ReadFile(audioPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestWhisperReadyStubEnv(t *testing.T) {
	stub := filepath.Join(t.TempDir(), "stub.sh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho stub-text\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NJ_WHISPER_CMD", stub)
	w := NewWhisperTranscriber()
	ready, hint := w.Ready()
	if !ready {
		t.Fatalf("expected ready, hint=%s", hint)
	}
	text, err := w.Transcribe(context.Background(), filepath.Join(t.TempDir(), "a.wav"))
	if err != nil {
		t.Fatal(err)
	}
	if text != "stub-text" {
		t.Fatalf("got %q", text)
	}
}

func TestWhisperMissingModel(t *testing.T) {
	t.Setenv("NJ_WHISPER_CMD", "")
	t.Setenv("NJ_WHISPER_MODEL", filepath.Join(t.TempDir(), "missing.bin"))
	w := &WhisperTranscriber{Cfg: WhisperConfig{
		Command:   filepath.Join(t.TempDir(), "also-missing"),
		ModelPath: filepath.Join(t.TempDir(), "missing.bin"),
	}}
	ready, hint := w.Ready()
	if ready {
		t.Fatal("expected not ready")
	}
	if hint == "" {
		t.Fatal("expected hint")
	}
}
