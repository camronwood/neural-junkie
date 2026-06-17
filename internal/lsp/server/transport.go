package server

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

// inboundMessage is either a response or notification from LSP server stdout.
type inboundMessage struct {
	ID       *int64          `json:"id"`
	Method   string          `json:"method"`
	Result   json.RawMessage `json:"result"`
	Error    *JSONRPCError   `json:"error"`
	Params   json.RawMessage `json:"params"`
	JSONRPC  string          `json:"jsonrpc"`
}

func (s *Session) startReader() {
	go func() {
		for {
			payload, err := readFramedPayload(s.stdout)
			if err != nil {
				return
			}
			var msg inboundMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}
			if msg.ID != nil {
				s.pendingMu.Lock()
				ch := s.pending[*msg.ID]
				delete(s.pending, *msg.ID)
				s.pendingMu.Unlock()
				if ch != nil {
					resp := JSONRPCResponse{JSONRPC: "2.0", ID: *msg.ID, Result: msg.Result, Error: msg.Error}
					out, _ := json.Marshal(resp)
					ch <- out
					close(ch)
				}
				continue
			}
			if msg.Method != "" {
				s.subMu.RLock()
			subs := append([]chan json.RawMessage(nil), s.subscribers...)
				s.subMu.RUnlock()
				for _, ch := range subs {
					select {
					case ch <- payload:
					default:
					}
				}
			}
		}
	}()
}

func readFramedPayload(r *bufio.Reader) ([]byte, error) {
	return ReadFramed(r)
}

// Subscribe returns a channel of raw JSON-RPC messages (notifications) from the language server.
func (s *Session) Subscribe() chan json.RawMessage {
	ch := make(chan json.RawMessage, 32)
	s.subMu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscription channel.
func (s *Session) Unsubscribe(ch chan json.RawMessage) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	var next []chan json.RawMessage
	for _, c := range s.subscribers {
		if c != ch {
			next = append(next, c)
		}
	}
	s.subscribers = next
	close(ch)
}

// Notify sends a JSON-RPC notification (no response expected).
func (s *Session) Notify(method string, params interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, err := json.Marshal(JSONRPCRequest{JSONRPC: "2.0", Method: method, Params: params})
	if err != nil {
		return err
	}
	return WriteFramed(s.stdin, body)
}

// Request sends a JSON-RPC request and waits for the matching response.
func (s *Session) Request(method string, params interface{}) (json.RawMessage, error) {
	s.mu.Lock()
	id := s.nextID.Add(1)
	ch := make(chan json.RawMessage, 1)
	s.pendingMu.Lock()
	if s.pending == nil {
		s.pending = make(map[int64]chan json.RawMessage)
	}
	s.pending[id] = ch
	s.pendingMu.Unlock()
	body, err := json.Marshal(JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if err := WriteFramed(s.stdin, body); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()
	raw := <-ch
	if len(raw) == 0 {
		return nil, io.EOF
	}
	var resp JSONRPCResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}

// RelayWebSocket bidirectionally relays JSON-RPC between WebSocket and the language server.
func (s *Session) RelayWebSocket(read func() ([]byte, error), write func([]byte) error) error {
	notifyCh := s.Subscribe()
	defer s.Unsubscribe(notifyCh)
	errCh := make(chan error, 2)
	go func() {
		for {
			data, err := read()
			if err != nil {
				errCh <- err
				return
			}
			var req JSONRPCRequest
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			if req.Method == "" {
				continue
			}
			if req.ID == 0 && req.Method != "" {
				// notification from client
				_ = s.Notify(req.Method, req.Params)
				continue
			}
			if req.ID != 0 {
				result, err := s.Request(req.Method, req.Params)
				resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID}
				if err != nil {
					if je, ok := err.(*JSONRPCError); ok {
						resp.Error = je
					} else {
						resp.Error = &JSONRPCError{Code: -32603, Message: err.Error()}
					}
				} else {
					resp.Result = result
				}
				out, _ := json.Marshal(resp)
				if err := write(out); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()
	go func() {
		for payload := range notifyCh {
			if err := write(payload); err != nil {
				errCh <- err
				return
			}
		}
	}()
	return <-errCh
}

var _ = sync.Mutex{}
