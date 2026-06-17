package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handleSidecarLSPRequest(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if *token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != *token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		var body struct {
			Lang   string          `json:"lang"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			URI    string          `json:"uri"`
			Text   string          `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sess, err := sidecarLSPManager.GetOrStart(r.Context(), "sidecar", body.Lang, root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if body.Text != "" && body.URI != "" {
			_ = sess.DidOpen(body.URI, body.Lang, body.Text)
		}
		var params interface{}
		if len(body.Params) > 0 {
			_ = json.Unmarshal(body.Params, &params)
		}
		result, err := sess.Request(body.Method, params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(result)
	}
}

func handleSidecarLSPHover(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if *token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != *token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		var body struct {
			Lang      string `json:"lang"`
			URI       string `json:"uri"`
			Text      string `json:"text"`
			Line      int    `json:"line"`
			Character int    `json:"character"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sess, err := sidecarLSPManager.GetOrStart(r.Context(), "sidecar", body.Lang, root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if body.Text != "" && body.URI != "" {
			_ = sess.DidOpen(body.URI, body.Lang, body.Text)
		}
		result, err := sess.Hover(body.URI, body.Line, body.Character)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(result)
	}
}
