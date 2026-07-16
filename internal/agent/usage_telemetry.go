package agent

import (
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/inference"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (a *Agent) collectTurnUsage(st *turnState) ai.InferenceUsage {
	if st == nil {
		return ai.InferenceUsage{}
	}
	if st.implSessionOutcome != nil {
		if raw, ok := st.implSessionOutcome["inference_usage"].(map[string]interface{}); ok {
			return usageFromMap(raw)
		}
	}
	if st.eff != nil {
		if ua, ok := st.eff.(ai.UsageAware); ok {
			return ua.TakeSessionUsage()
		}
	}
	return ai.InferenceUsage{}
}

func usageFromMap(raw map[string]interface{}) ai.InferenceUsage {
	u := ai.InferenceUsage{}
	if v, ok := raw["prompt_tokens"].(float64); ok {
		u.PromptTokens = int(v)
	} else if v, ok := raw["prompt_tokens"].(int); ok {
		u.PromptTokens = v
	}
	if v, ok := raw["completion_tokens"].(float64); ok {
		u.CompletionTokens = int(v)
	} else if v, ok := raw["completion_tokens"].(int); ok {
		u.CompletionTokens = v
	}
	if v, ok := raw["calls"].(float64); ok {
		u.Calls = int(v)
	} else if v, ok := raw["calls"].(int); ok {
		u.Calls = v
	}
	return u
}

func (a *Agent) applyUsageTelemetry(st *turnState) {
	if st == nil || st.msg == nil || st.responseMsg == nil {
		return
	}
	u := a.collectTurnUsage(st)
	usageMap := ai.MapUsage(u)
	if usageMap == nil {
		return
	}

	snap := a.LastRoutingSnapshot()
	if snap.ProviderID != "" {
		usageMap["provider_id"] = snap.ProviderID
	}
	if snap.ChatModel != "" {
		usageMap["model"] = snap.ChatModel
	}
	if snap.CostTier != "" {
		usageMap["cost_tier"] = snap.CostTier
	}

	costUSD := ai.EstimateCostUSD(snap.ProviderID, snap.ChatModel, snap.CostTier, u.PromptTokens, u.CompletionTokens)
	if costUSD > 0 {
		usageMap["estimated_cost_usd"] = costUSD
	}

	protocol.ApplyInferenceUsageMeta(st.responseMsg, usageMap)
	a.sendTelemetryEvent(st.msg, "usage", usageMap)

	if store := inference.DefaultStore(); store != nil {
		store.Record(inference.TurnRecordFromUsage(
			st.msg.Channel,
			a.Info.ID,
			a.Info.Name,
			snap.ProviderID,
			snap.ChatModel,
			snap.CostTier,
			u,
			costUSD,
		))
	}
}
