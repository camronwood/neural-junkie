// tool-approval-hook is a Gemini CLI BeforeTool hook that routes tool approval
// decisions through the Neural Junkie server. Gemini spawns this binary before
// each tool call. It reads the tool details from stdin, POSTs them to the
// server's /api/tool-approvals endpoint (which blocks until the user decides),
// and writes the decision to stdout in the format Gemini expects.
//
// Usage in Gemini settings.json:
//
//	"hooks": {
//	  "BeforeTool": [{
//	    "hooks": [{
//	      "type": "command",
//	      "command": "tool-approval-hook --server http://localhost:8080 --agent Gemini",
//	      "timeout": 180000
//	    }]
//	  }]
//	}
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type hookInput struct {
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	CWD           string                 `json:"cwd"`
	Timestamp     string                 `json:"timestamp"`
}

type approvalRequest struct {
	AgentID   string                 `json:"agent_id"`
	AgentName string                 `json:"agent_name"`
	SessionID string                 `json:"session_id"`
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
	Channel   string                 `json:"channel"`
	Mode      string                 `json:"mode"`
}

type approvalResponse struct {
	Status   string `json:"status"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type hookOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "Neural Junkie server URL")
	agentName := flag.String("agent", "Gemini", "Agent display name")
	agentID := flag.String("agent-id", "gemini-cli", "Agent ID")
	channel := flag.String("channel", "", "Chat channel for approval messages")
	mode := flag.String("mode", "interactive", "Approval mode: interactive, auto_edit, yolo")
	flag.Parse()

	// Read hook input from stdin
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	var input hookInput
	if err := json.Unmarshal(inputData, &input); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse hook input: %v\n", err)
		os.Exit(1)
	}

	resolvedChannel := *channel
	if resolvedChannel == "" {
		resolvedChannel = os.Getenv("NEURAL_JUNKIE_CHANNEL")
	}
	if resolvedChannel == "" {
		resolvedChannel = "general"
	}

	// Build request to Neural Junkie server
	req := approvalRequest{
		AgentID:   *agentID,
		AgentName: *agentName,
		SessionID: input.SessionID,
		ToolName:  input.ToolName,
		ToolInput: input.ToolInput,
		Channel:   resolvedChannel,
		Mode:      *mode,
	}

	body, _ := json.Marshal(req)

	client := &http.Client{Timeout: 200 * time.Second}
	resp, err := client.Post(*serverURL+"/api/tool-approvals", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to contact server: %v\n", err)
		// On network error, allow the tool (fail-open so Gemini doesn't hang)
		writeAllow()
		return
	}
	defer resp.Body.Close()

	var result approvalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse server response: %v\n", err)
		writeAllow()
		return
	}

	if result.Decision == "allow" {
		writeAllow()
	} else {
		reason := result.Reason
		if reason == "" {
			reason = "Tool call rejected by user"
		}
		writeDeny(reason)
	}
}

func writeAllow() {
	out := hookOutput{Decision: "allow"}
	json.NewEncoder(os.Stdout).Encode(out)
	os.Exit(0)
}

func writeDeny(reason string) {
	out := hookOutput{Decision: "deny", Reason: reason}
	json.NewEncoder(os.Stdout).Encode(out)
	fmt.Fprint(os.Stderr, reason)
	os.Exit(2)
}
