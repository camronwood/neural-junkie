package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testStructuredSchema = json.RawMessage(`{
	"type": "object",
	"properties": {"answer": {"type": "string"}},
	"required": ["answer"]
}`)

func TestOllamaStructuredOutputSerializesFormat(t *testing.T) {
	tests := []struct {
		name       string
		schema     json.RawMessage
		wantFormat any
	}{
		{name: "json", wantFormat: "json"},
		{name: "schema", schema: testStructuredSchema, wantFormat: map[string]any{"type": "object"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": map[string]any{"content": `{"answer":"ok"}`},
					"done":    true,
				})
			}))
			defer srv.Close()

			provider := NewOllamaProviderWithConfig(srv.URL, "test-model")
			result, err := provider.GenerateStructuredResponse(context.Background(), StructuredOutputRequest{
				Prompt:     "answer",
				SchemaName: "answer",
				JSONSchema: tc.schema,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Content != `{"answer":"ok"}` {
				t.Fatalf("content = %q", result.Content)
			}
			if tc.name == "json" {
				if body["format"] != tc.wantFormat {
					t.Fatalf("format = %#v, want %#v", body["format"], tc.wantFormat)
				}
				return
			}
			format, ok := body["format"].(map[string]any)
			if !ok || format["type"] != "object" {
				t.Fatalf("format = %#v, want JSON Schema object", body["format"])
			}
		})
	}
}

func TestOpenAICompatibleStructuredOutputSerializesResponseFormat(t *testing.T) {
	tests := []struct {
		name     string
		provider func(endpoint string) StructuredOutputProvider
	}{
		{
			name: "openai compatible",
			provider: func(endpoint string) StructuredOutputProvider {
				return NewOpenAICompatProvider(endpoint+"/v1", "", "test-model", nil)
			},
		},
		{
			name: "lm studio",
			provider: func(endpoint string) StructuredOutputProvider {
				return NewLMStudioProviderWithConfig(endpoint+"/v1", "test-model")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"choices": []map[string]any{{
						"message": map[string]any{"content": `{"answer":"ok"}`},
					}},
				})
			}))
			defer srv.Close()

			result, err := tc.provider(srv.URL).GenerateStructuredResponse(context.Background(), StructuredOutputRequest{
				Prompt:     "answer",
				SchemaName: "answer_schema",
				JSONSchema: testStructuredSchema,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Content != `{"answer":"ok"}` {
				t.Fatalf("content = %q", result.Content)
			}
			format, ok := body["response_format"].(map[string]any)
			if !ok || format["type"] != "json_schema" {
				t.Fatalf("response_format = %#v", body["response_format"])
			}
			schema, ok := format["json_schema"].(map[string]any)
			if !ok || schema["name"] != "answer_schema" || schema["strict"] != true {
				t.Fatalf("json_schema = %#v", format["json_schema"])
			}
			if _, ok := schema["schema"].(map[string]any); !ok {
				t.Fatalf("schema = %#v, want object", schema["schema"])
			}
		})
	}
}
