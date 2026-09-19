package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func decodeDevCompleteRequest(r *http.Request) (DevCompleteRequest, error) {
	var req DevCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	return req, nil
}

func handleDevComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
		return
	}
	req, err := decodeDevCompleteRequest(r)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prefix) == "" {
		http.Error(w, "prefix required", http.StatusBadRequest)
		return
	}

	model := resolveDevCompleteModel(req.Model)
	endpoint := resolveOllamaEndpoint()
	prompt := AssembleDevCompletePrompt(req)
	n := clampDevCompleteN(req.NRequests, 2)

	completions := make([]string, 0, n)
	temps := []float64{devCompletePrimaryTemp, devCompleteAlternateTemp}
	predicts := []int{devCompletePrimaryPredict, devCompleteAlternatePredict}
	for i := 0; i < n; i++ {
		text, err := ollamaGenerateOnce(r.Context(), endpoint, model, prompt, predicts[i], temps[i], 6*time.Second)
		if err != nil {
			if i == 0 {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			break
		}
		completions = append(completions, text)
	}
	completions = dedupeCompletions(completions)
	primary := ""
	if len(completions) > 0 {
		primary = completions[0]
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"completion":  primary,
		"completions": completions,
		"model":       model,
	})
}

// handleDevCompleteStream streams NDJSON completion chunks from Ollama (stream:true).
// Lines: {"type":"chunk","text":"..."} then {"type":"done","completion":"...","completions":[...]}.
func handleDevCompleteStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
		return
	}
	req, err := decodeDevCompleteRequest(r)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prefix) == "" {
		http.Error(w, "prefix required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	model := resolveDevCompleteModel(req.Model)
	endpoint := resolveOllamaEndpoint()
	prompt := AssembleDevCompletePrompt(req)
	wantAlternate := clampDevCompleteN(req.NRequests, 2) > 1

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	writeEvent := func(v interface{}) {
		raw, mErr := json.Marshal(v)
		if mErr != nil {
			return
		}
		_, _ = w.Write(raw)
		_, _ = w.Write([]byte("\n"))
		flusher.Flush()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	primary, err := ollamaGenerateStream(ctx, endpoint, model, prompt, devCompletePrimaryPredict, devCompletePrimaryTemp, func(chunk string) {
		if chunk == "" {
			return
		}
		writeEvent(map[string]string{"type": "chunk", "text": chunk})
	})
	if err != nil {
		writeEvent(map[string]string{"type": "error", "error": err.Error()})
		return
	}

	completions := []string{primary}
	if wantAlternate {
		alt, altErr := ollamaGenerateOnce(ctx, endpoint, model, prompt, devCompleteAlternatePredict, devCompleteAlternateTemp, 5*time.Second)
		if altErr == nil {
			completions = append(completions, alt)
		}
	}
	completions = dedupeCompletions(completions)
	primaryOut := ""
	if len(completions) > 0 {
		primaryOut = completions[0]
	}
	writeEvent(map[string]interface{}{
		"type":        "done",
		"completion":  primaryOut,
		"completions": completions,
		"model":       model,
	})
}

func ollamaGenerateOnce(parent context.Context, endpoint, model, prompt string, numPredict int, temperature float64, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	body := ollamaGenerateBody(model, prompt, false, numPredict, temperature)
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/generate", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = fmt.Sprintf("ollama status %d", resp.StatusCode)
		}
		return "", fmt.Errorf("%s", msg)
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("invalid ollama response")
	}
	return strings.TrimSpace(out.Response), nil
}

func ollamaGenerateStream(
	ctx context.Context,
	endpoint, model, prompt string,
	numPredict int,
	temperature float64,
	onChunk func(string),
) (string, error) {
	body := ollamaGenerateBody(model, prompt, true, numPredict, temperature)
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/generate", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = fmt.Sprintf("ollama status %d", resp.StatusCode)
		}
		return "", fmt.Errorf("%s", msg)
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		if chunk.Response != "" {
			full.WriteString(chunk.Response)
			onChunk(chunk.Response)
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return strings.TrimSpace(full.String()), err
	}
	return strings.TrimSpace(full.String()), nil
}
