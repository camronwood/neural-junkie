package agent

import "testing"

func TestFormatRoutingThinkingDetail(t *testing.T) {
	detail := formatRoutingThinkingDetailForTest(RoutingSnapshot{
		ChatModel: "qwen3.5:9b",
		ToolModel: "qwen2.5:7b",
		Reason:    "tool_fallback",
	})
	if detail == "" {
		t.Fatal("expected non-empty detail")
	}
	if want := "chat: qwen3.5:9b · tools: qwen2.5:7b (tool_fallback)"; detail != want {
		t.Fatalf("detail = %q, want %q", detail, want)
	}
}

func TestFormatRoutingThinkingDetailChatOnly(t *testing.T) {
	detail := formatRoutingThinkingDetailForTest(RoutingSnapshot{
		ChatModel: "llama3.1:8b",
		Reason:    "capability_routing",
	})
	if want := "chat: llama3.1:8b (capability_routing)"; detail != want {
		t.Fatalf("detail = %q, want %q", detail, want)
	}
}

func TestFormatRoutingThinkingDetailReactTools(t *testing.T) {
	detail := formatRoutingThinkingDetailForTest(RoutingSnapshot{
		ChatModel: "gemma3:12b",
		ToolModel: "gemma3:12b",
		Reason:    "react_tools",
	})
	if want := "chat: gemma3:12b (react_tools)"; detail != want {
		t.Fatalf("detail = %q, want %q", detail, want)
	}
}
