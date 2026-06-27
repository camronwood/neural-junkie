package music

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSidecarGeneratorGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/music/generate" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mime":"audio/wav","data":"` + base64.StdEncoding.EncodeToString([]byte("RIFF")) + `"}`))
	}))
	t.Cleanup(srv.Close)

	gen := NewSidecarGenerator(func() string { return srv.URL })
	mime, b64, err := gen.Generate(context.Background(), Request{StyleTags: "lo-fi", Lyrics: "[Instrumental]", DurationSec: 10})
	if err != nil {
		t.Fatal(err)
	}
	if mime != "audio/wav" {
		t.Fatalf("mime = %q", mime)
	}
	if b64 == "" {
		t.Fatal("expected data")
	}
}
