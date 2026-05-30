package main

import (
	"net/http"
)

const (
	capLoRATraining = "lora-training"
	capLoRACompose  = "lora-compose"
	capLoRAAdapters = "lora-adapters"
)

func hasLoRACapability(cap string) bool {
	return appConfig != nil && appConfig.AnyPackCapability(cap)
}

func requireLoRACapability(w http.ResponseWriter, cap string) bool {
	if hasLoRACapability(cap) {
		return true
	}
	msg := "Specialist tuning pack required — enable it in Settings → Domain packs"
	switch cap {
	case capLoRATraining:
		msg = "Specialist tuning pack required for LoRA training"
	case capLoRACompose:
		msg = "Specialist tuning pack required for LoRA compose"
	case capLoRAAdapters:
		msg = "Specialist tuning pack required for pack LoRA install"
	}
	http.Error(w, msg, http.StatusForbidden)
	return false
}
