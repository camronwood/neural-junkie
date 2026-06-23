// Package server manages persistent LSP language server processes.
package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
)

// JSONRPCRequest is a minimal LSP JSON-RPC request.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse is a minimal LSP JSON-RPC response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError is an LSP error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *JSONRPCError) Error() string { return e.Message }

// Session wraps a language server subprocess.
type Session struct {
	lang          string
	root          string
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        *bufio.Reader
	mu            sync.Mutex
	nextID        atomic.Int64
	initialized   bool
	pending       map[int64]chan json.RawMessage
	pendingMu     sync.Mutex
	subscribers   []chan json.RawMessage
	subMu         sync.RWMutex
}

// Manager owns LSP sessions keyed by workspace+language.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewManager creates an LSP session manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

func sessionKey(workspaceID, lang string) string {
	return workspaceID + ":" + lang
}

// languageServerCommand returns the command to start a language server.
func languageServerCommand(lang string) (string, []string, error) {
	switch lang {
	case "go":
		if _, err := exec.LookPath("gopls"); err != nil {
			return "", nil, fmt.Errorf("gopls not found on PATH")
		}
		return "gopls", []string{"-mode=stdio"}, nil
	case "rust":
		if _, err := exec.LookPath("rust-analyzer"); err != nil {
			return "", nil, fmt.Errorf("rust-analyzer not found on PATH")
		}
		return "rust-analyzer", nil, nil
	case "python":
		if _, err := exec.LookPath("pyright-langserver"); err != nil {
			if _, err2 := exec.LookPath("pyright"); err2 != nil {
				return "", nil, fmt.Errorf("pyright-langserver not found on PATH")
			}
			return "pyright", []string{"--stdio"}, nil
		}
		return "pyright-langserver", []string{"--stdio"}, nil
	case "typescript", "javascript":
		if _, err := exec.LookPath("typescript-language-server"); err != nil {
			return "", nil, fmt.Errorf("typescript-language-server not found on PATH")
		}
		return "typescript-language-server", []string{"--stdio"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported language %q", lang)
	}
}

// GetOrStart returns an initialized session for workspace+lang.
func (m *Manager) GetOrStart(ctx context.Context, workspaceID, lang, root string) (*Session, error) {
	key := sessionKey(workspaceID, lang)
	m.mu.Lock()
	if s, ok := m.sessions[key]; ok && s.cmd != nil && s.cmd.Process != nil {
		m.mu.Unlock()
		return s, nil
	}
	m.mu.Unlock()

	s, err := startSession(ctx, lang, root)
	if err != nil {
		return nil, err
	}
	if err := s.initialize(root); err != nil {
		s.Close()
		return nil, err
	}
	m.mu.Lock()
	m.sessions[key] = s
	m.mu.Unlock()
	return s, nil
}

func startSession(ctx context.Context, lang, root string) (*Session, error) {
	bin, args, err := languageServerCommand(lang)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = root
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sess := &Session{
		lang:    lang,
		root:    root,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdoutPipe),
		pending: make(map[int64]chan json.RawMessage),
	}
	sess.startReader()
	return sess, nil
}

func (s *Session) initialize(root string) error {
	rootURI := "file://" + strings.TrimPrefix(root, "/")
	if !strings.HasPrefix(rootURI, "file://") {
		rootURI = "file:///" + strings.TrimPrefix(root, "/")
	}
	params := map[string]interface{}{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]interface{}{},
		"workspaceFolders": []map[string]string{
			{"uri": rootURI, "name": "workspace"},
		},
	}
	_, err := s.Request("initialize", params)
	if err != nil {
		return err
	}
	err = s.Notify("initialized", map[string]interface{}{})
	if err != nil {
		return err
	}
	s.initialized = true
	return nil
}

// Call sends a JSON-RPC request and waits for the matching response.
func (s *Session) Call(method string, params interface{}) (json.RawMessage, error) {
	return s.Request(method, params)
}

// DidOpen notifies the server a document was opened.
func (s *Session) DidOpen(uri, languageID, text string) error {
	return s.Notify("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       text,
		},
	})
}

// DidChange sends full-buffer document changes.
func (s *Session) DidChange(uri string, version int, text string) error {
	return s.Notify("textDocument/didChange", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri, "version": version},
		"contentChanges": []map[string]interface{}{
			{"text": text},
		},
	})
}

// DidClose notifies document close.
func (s *Session) DidClose(uri string) error {
	return s.Notify("textDocument/didClose", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
	})
}

// Hover requests hover information at a position.
func (s *Session) Hover(uri string, line, character int) (json.RawMessage, error) {
	return s.Request("textDocument/hover", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]int{"line": line, "character": character},
	})
}

// Completion requests completions at a position.
func (s *Session) Completion(uri string, line, character int) (json.RawMessage, error) {
	return s.Request("textDocument/completion", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]int{"line": line, "character": character},
	})
}

// Definition requests go-to-definition.
func (s *Session) Definition(uri string, line, character int) (json.RawMessage, error) {
	return s.Request("textDocument/definition", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]int{"line": line, "character": character},
	})
}

// References requests find references.
func (s *Session) References(uri string, line, character int) (json.RawMessage, error) {
	return s.Request("textDocument/references", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]int{"line": line, "character": character},
		"context":      map[string]bool{"includeDeclaration": true},
	})
}

// Rename requests a workspace rename.
func (s *Session) Rename(uri string, line, character int, newName string) (json.RawMessage, error) {
	return s.Request("textDocument/rename", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]int{"line": line, "character": character},
		"newName":      newName,
	})
}

// Close shuts down the language server.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stdin != nil {
		_, _ = s.Call("shutdown", nil)
		_, _ = s.Call("exit", nil)
		_ = s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

// EncodeMessage frames a JSON-RPC message for WebSocket transport.
func EncodeMessage(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// DecodeMessage parses a JSON-RPC message from the client.
func DecodeMessage(data []byte) (JSONRPCRequest, error) {
	var req JSONRPCRequest
	err := json.Unmarshal(data, &req)
	return req, err
}

// WriteFramed writes LSP stdio framing to w.
func WriteFramed(w io.Writer, payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadFramed reads one LSP-framed message.
func ReadFramed(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "Content-Length:") {
		return nil, fmt.Errorf("expected Content-Length header, got %q", line)
	}
	var length int
	fmt.Sscanf(strings.TrimSpace(line), "Content-Length: %d", &length)
	if _, err := r.ReadString('\n'); err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	_, err = io.ReadFull(r, buf)
	return buf, err
}

// ProxyClientToServer copies framed messages from client to server stdin.
func ProxyClientToServer(client io.Reader, serverStdin io.Writer) error {
	r := bufio.NewReader(client)
	for {
		payload, err := ReadFramed(r)
		if err != nil {
			return err
		}
		if err := WriteFramed(serverStdin, payload); err != nil {
			return err
		}
	}
}

// ProxyServerToClient copies framed messages from server stdout to client.
func ProxyServerToClient(serverStdout io.Reader, client io.Writer) error {
	r := bufio.NewReader(serverStdout)
	for {
		payload, err := ReadFramed(r)
		if err != nil {
			return err
		}
		if _, err := client.Write(payload); err != nil {
			return err
		}
		// Also write length header for websocket clients expecting raw JSON
		_ = bytes.NewReader(payload)
	}
}
