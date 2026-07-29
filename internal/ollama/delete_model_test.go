package ollama

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeleteModel_usesDELETE(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.URL.Path != "/api/delete" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"demo:latest"`) {
			t.Fatalf("body = %s", body)
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("405 method not allowed"))
	}))
	t.Cleanup(srv.Close)

	m := NewManager(srv.URL)
	if err := m.DeleteModel(context.Background(), "demo:latest"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 1 || methods[0] != http.MethodDelete {
		t.Fatalf("methods = %v, want [DELETE]", methods)
	}
}

func TestDeleteModel_fallsBackToPOST(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("405 method not allowed"))
	}))
	t.Cleanup(srv.Close)

	m := NewManager(srv.URL)
	if err := m.DeleteModel(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != http.MethodDelete || methods[1] != http.MethodPost {
		t.Fatalf("methods = %v, want [DELETE POST]", methods)
	}
}
