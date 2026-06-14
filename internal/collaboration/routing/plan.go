package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/config"
	unified "github.com/camronwood/neural-junkie/internal/routing"
)

// PlanInput is the static context for predicting collaboration task routing.
type PlanInput struct {
	TaskText                string
	AgentName               string
	AgentType               string
	AgentModel              string
	DefaultProviderID       string
	TaskProviderOverride    string
	TaskOllamaModelOverride  string
	SmartRoutingEnabled     bool
	HasLoRACapability       bool
	HasUserImages           bool
	SupportsVision          bool
	Providers               []config.ProviderConfig
	InstalledLoRATags       map[string]struct{}
	InstalledOllamaTags     map[string]struct{}
}

// PlanResult predicts provider and model selection for one collaboration task.
type PlanResult struct {
	ProviderID     string
	ProviderReason string
	OllamaModel    string
	ModelReason    string
}

// PlanTask mirrors collaboration execution routing heuristics without calling providers.
func PlanTask(in PlanInput) PlanResult {
	providerID := strings.TrimSpace(in.DefaultProviderID)
	providerReason := "default_agent_provider"

	if override := strings.TrimSpace(in.TaskProviderOverride); override != "" {
		providerID = override
		providerReason = "task_provider_metadata"
	} else if in.SmartRoutingEnabled {
		dec := unified.ClassifyRules(unified.Input{
			Text:          in.TaskText,
			AgentType:     in.AgentType,
			AgentModel:    in.AgentModel,
			HasUserImages: in.HasUserImages,
			InstalledTags: in.InstalledLoRATags,
		})
		selID, reason := unified.PickProviderID(unified.ProviderPickInput{
			Decision:          dec,
			HasUserImages:     in.HasUserImages,
			Providers:         in.Providers,
			DefaultProviderID: in.DefaultProviderID,
			InstalledTags:     in.InstalledLoRATags,
		})
		if strings.TrimSpace(selID) != "" {
			providerID = selID
			providerReason = reason
		}
	}

	if override := strings.TrimSpace(in.TaskOllamaModelOverride); override != "" {
		return PlanResult{
			ProviderID:     providerID,
			ProviderReason: providerReason,
			OllamaModel:    override,
			ModelReason:    "task_ollama_model",
		}
	}

	if in.HasLoRACapability {
		dec := unified.ClassifyRules(unified.Input{
			Text:          in.TaskText,
			AgentType:     in.AgentType,
			AgentModel:    in.AgentModel,
			InstalledTags: in.InstalledLoRATags,
		})
		tag, tagReason := dec.LoRATag, dec.Reason
		if tag == "" {
			tag, tagReason = SelectComposedTag(LoRAInput{
				TaskText:      in.TaskText,
				AgentType:     in.AgentType,
				AgentModel:    in.AgentModel,
				InstalledTags: in.InstalledLoRATags,
			})
		}
		if tag != "" {
			return PlanResult{
				ProviderID:     providerID,
				ProviderReason: providerReason,
				OllamaModel:    tag,
				ModelReason:    tagReason,
			}
		}
	}

	if !keepAgentModelForCollabTask(in.TaskText) && LooksLightweightCollabTask(in.TaskText) {
		tag, reason := SelectLightOllamaTag(in.InstalledOllamaTags)
		if tag != "" {
			return PlanResult{
				ProviderID:     providerID,
				ProviderReason: providerReason,
				OllamaModel:    tag,
				ModelReason:    reason,
			}
		}
	}

	modelReason := "agent_default_model"
	if keepAgentModelForCollabTask(in.TaskText) && LooksLightweightCollabTask(in.TaskText) {
		modelReason = "deliverable_task_keep_agent_model"
	}
	return PlanResult{
		ProviderID:     providerID,
		ProviderReason: providerReason,
		OllamaModel:    strings.TrimSpace(in.AgentModel),
		ModelReason:    modelReason,
	}
}

func isArchitectureDocDeliverable(taskText string) bool {
	lower := strings.ToLower(taskText)
	return strings.Contains(lower, "frontend_architecture_plan.md") ||
		strings.Contains(lower, "findings.md")
}

// keepAgentModelForCollabTask skips light local model downgrades for tasks that need full agent quality.
func keepAgentModelForCollabTask(taskText string) bool {
	taskText = strings.TrimSpace(taskText)
	if taskText == "" {
		return false
	}
	if isArchitectureDocDeliverable(taskText) {
		return false
	}
	task := collaboration.CollaborationTask{Title: taskText, Description: taskText}
	if collaboration.TaskRequiresFileDeliverable(task) {
		return true
	}
	if synthesisKeywords(strings.ToLower(taskText)) {
		return true
	}
	return false
}

// ExpectedModel returns the model label to show for a planned route.
func (r PlanResult) ExpectedModel(providers []config.ProviderConfig) string {
	if !strings.EqualFold(providerType(providers, r.ProviderID), "ollama") {
		return ""
	}
	return strings.TrimSpace(r.OllamaModel)
}

// RoutingReason returns the primary reason code for UI/logging.
func (r PlanResult) RoutingReason() string {
	if r.ModelReason != "" && r.ModelReason != "agent_default_model" {
		return r.ModelReason
	}
	return r.ProviderReason
}

func providerType(providers []config.ProviderConfig, id string) string {
	id = strings.TrimSpace(id)
	for _, p := range providers {
		if strings.TrimSpace(p.ID) == id {
			return strings.TrimSpace(p.Type)
		}
	}
	return ""
}
