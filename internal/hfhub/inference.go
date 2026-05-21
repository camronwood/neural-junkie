package hfhub

import (
	"fmt"
	"strings"
)

// HFInferenceBaseURL is the serverless HF Inference provider (replaces api-inference.huggingface.co).
const HFInferenceBaseURL = "https://router.huggingface.co/hf-inference"

// InferenceModelURL returns the POST URL for a Hub model id on HF Inference.
func InferenceModelURL(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	return fmt.Sprintf("%s/models/%s", strings.TrimRight(HFInferenceBaseURL, "/"), modelID)
}
