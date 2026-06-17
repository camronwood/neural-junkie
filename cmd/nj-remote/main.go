package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/pathutil"
)

var (
	listenAddr = flag.String("addr", ":19876", "HTTP listen address")
	root       = flag.String("root", ".", "Workspace root directory")
	token      = flag.String("token", "", "Bearer token (optional)")
)

func main() {
	flag.Parse()
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(absRoot)
	if err != nil || !info.IsDir() {
		log.Fatalf("root must be a directory: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/fs/list", auth(handleList(absRoot)))
	mux.HandleFunc("/api/fs/read", auth(handleRead(absRoot)))
	mux.HandleFunc("/api/fs/write", auth(handleWrite(absRoot)))
	mux.HandleFunc("/api/fs/stat", auth(handleStat(absRoot)))
	mux.HandleFunc("/api/exec", auth(handleExec(absRoot)))
	mux.HandleFunc("/api/pty/ws", handleSidecarPTY)

	log.Printf("nj-remote listening on %s root=%s", *listenAddr, absRoot)
	log.Fatal(http.ListenAndServe(*listenAddr, mux))
}

func auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if *token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != *token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func relPath(root, query string) (string, error) {
	rel := strings.TrimPrefix(query, "/")
	full := filepath.Join(root, rel)
	return pathutil.WithinRoot(root, full)
}

func handleList(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		abs, err := relPath(root, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		base := strings.TrimPrefix(filepath.ToSlash(r.URL.Query().Get("path")), "/")
		var out []map[string]interface{}
		for _, e := range entries {
			info, _ := e.Info()
			p := e.Name()
			if base != "" && base != "." {
				p = base + "/" + e.Name()
			}
			item := map[string]interface{}{
				"name":   e.Name(),
				"path":   p,
				"is_dir": e.IsDir(),
			}
			if info != nil {
				item["size"] = info.Size()
				item["mod_time"] = info.ModTime().Format(time.RFC3339)
			}
			out = append(out, item)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": out})
	}
}

func handleRead(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		abs, err := relPath(root, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"content": string(data)})
	}
}

func handleWrite(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		abs, err := relPath(root, body.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(abs, []byte(body.Content), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func handleStat(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		abs, err := relPath(root, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		info, err := os.Stat(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":     info.Name(),
			"size":     info.Size(),
			"is_dir":   info.IsDir(),
			"mod_time": info.ModTime().Format(time.RFC3339),
		})
	}
}

func handleExec(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
			Cwd     string   `json:"cwd"`
			Env     []string `json:"env"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cwd := root
		if body.Cwd != "" {
			abs, err := relPath(root, body.Cwd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			cwd = abs
		}
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, body.Command, body.Args...)
		cmd.Dir = cwd
		if len(body.Env) > 0 {
			cmd.Env = append(os.Environ(), body.Env...)
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		exitCode := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"stdout":    stdout.String(),
			"stderr":    stderr.String(),
			"exit_code": exitCode,
		})
	}
}
