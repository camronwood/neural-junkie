package hub

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidBootstrapTokenFromEnv(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN", "test-bootstrap-secret")
	r := httptest.NewRequest(http.MethodPost, "/api/auth/session", nil)
	r.Header.Set("X-NJ-Bootstrap", "test-bootstrap-secret")
	if !ValidBootstrapToken(r) {
		t.Fatal("expected valid bootstrap token")
	}
	r2 := httptest.NewRequest(http.MethodPost, "/api/auth/session", nil)
	r2.Header.Set("X-NJ-Bootstrap", "wrong")
	if ValidBootstrapToken(r2) {
		t.Fatal("expected invalid bootstrap token")
	}
}

func TestValidBootstrapTokenFromFile(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap.token")
	if err := os.WriteFile(path, []byte("file-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN_FILE", path)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/session", nil)
	r.Header.Set("Authorization", "Bearer file-secret")
	if !ValidBootstrapToken(r) {
		t.Fatal("expected bootstrap from file")
	}
}

func TestRequireSessionForMutationRelaxedLocalMember(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "1")
	sm := NewSessionManager()
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/send", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	sess, ok := RequireSessionForMutation(rec, r, sm)
	if !ok {
		t.Fatalf("expected relaxed local session, status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sess.Role != "member" {
		t.Fatalf("expected member role, got %q", sess.Role)
	}
}

func TestRequireSessionForMutationStrictLoopbackDenied(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	sm := NewSessionManager()
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/send", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	_, ok := RequireSessionForMutation(rec, r, sm)
	if ok {
		t.Fatal("expected strict loopback without session to be denied")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
