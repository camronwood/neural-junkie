package protocol

import "testing"

func TestLooksLikeStackToolCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"docker-compose up -d", true},
		{"npm install", true},
		{"kubectl get pods", true},
		{"make build", true},
		{"cat resource-api/json_endpoints/products.json", false},
		{"grep schema resource-api/", false},
	}
	for _, tc := range cases {
		if got := LooksLikeStackToolCommand(tc.cmd); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.cmd, got, tc.want)
		}
	}
}

func TestLooksLikeReadOnlyInspectionCommand(t *testing.T) {
	if !LooksLikeReadOnlyInspectionCommand("cat collabs/abc/out.md") {
		t.Fatal("expected cat to be read-only inspection")
	}
	if LooksLikeReadOnlyInspectionCommand("npm test") {
		t.Fatal("npm test should not be read-only inspection")
	}
}
