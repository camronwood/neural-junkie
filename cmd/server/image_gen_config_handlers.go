package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
)

func imageGenConfigPublic() map[string]interface{} {
	st := appConfig.ResolvedImageGen()
	cfg := ai.ImageGenConfigFromSettings(st)
	return map[string]interface{}{
		"provider":        st.Provider,
		"model":           st.Model,
		"ollama_model":    st.OllamaModel,
		"openai_base_url": st.OpenAIBaseURL,
		"keep_alive":      st.KeepAlive,
		"openai_key_set":  strings.TrimSpace(st.OpenAIAPIKey) != "",
		"runtime":         cfg,
	}
}

func handleImageGenConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st := appConfig.Redacted().ImageGen
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"provider":        st.Provider,
			"model":           st.Model,
			"ollama_model":    st.OllamaModel,
			"openai_base_url": appConfig.ImageGen.OpenAIBaseURL,
			"keep_alive":      st.KeepAlive,
			"openai_key_set":  appConfig.ImageGen.OpenAIAPIKey != "",
		})
	case http.MethodPut, http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var body config.ImageGenSettings
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if isRedactedPlaceholder(body.OpenAIAPIKey) {
			body.OpenAIAPIKey = appConfig.ImageGen.OpenAIAPIKey
		}
		appConfig.ImageGen = body
		if err := appConfig.Save(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		config.SetAppConfig(appConfig)
		writeJSON(w, http.StatusOK, imageGenConfigPublic())
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
