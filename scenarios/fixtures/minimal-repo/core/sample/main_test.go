package main

import (
	"io"
	"os"
	"testing"
)

func TestHelloWorld(t *testing.T) {
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	HelloWorld()

	w.Close()
	os.Stdout = stdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(out) != "Hello, World!\n" {
		t.Errorf("expected %q, got %q", "Hello, World!\n", out)
	}
}
