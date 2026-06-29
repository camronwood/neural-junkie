package main

import (
    "testing"
    "bytes"
    "os"
)

func TestHelloWorld(t *testing.T) {
    // Capture standard output
    stdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    // Call the HelloWorld function
    HelloWorld()

    // Restore standard output
    w.Close()
    os.Stdout = stdout

    // Read the output
    out, _ := bytes.ReadAll(r)
    if string(out) != "Hello, World!\n" {
        t.Errorf("Expected 'Hello, World!\\n', got '%s'", out)
    }
}