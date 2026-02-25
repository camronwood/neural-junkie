package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ansiRegex strips ANSI escape sequences (colors, cursor control, etc.)
// that leak through the PTY since the subprocess thinks it's a real terminal.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][0-9A-B]`)

var (
	// ErrCLIProviderTimeout marks timeout failures from CLI-backed providers.
	ErrCLIProviderTimeout = errors.New("cli provider timeout")
)

// CLIAgentProvider implements AIProvider by invoking a CLI-based AI agent as a subprocess.
// This is designed to be generic -- it can wrap any CLI agent tool (Cursor, Claude CLI, etc.)
// by configuring the command, args, and environment.
type CLIAgentProvider struct {
	// Command is the CLI binary to invoke (e.g. "agent" for Cursor CLI)
	Command string

	// BaseArgs are the default arguments appended before the prompt
	// (e.g. ["-p", "--output-format", "json"] for Cursor headless mode)
	BaseArgs []string

	// WorkDir is the working directory the CLI process runs in.
	// For Cursor CLI this determines which codebase the agent operates on.
	WorkDir string

	// Timeout is the maximum duration for a single invocation.
	// CLI agents can take 30-120s+ for complex tasks.
	Timeout time.Duration

	// Env holds extra environment variables passed to the subprocess
	// (e.g. CURSOR_API_KEY). Inherited env is always included.
	Env map[string]string

	// Model is the display name shown in the UI (e.g. "cursor-agent")
	Model string

	// ProviderName is the display name of the provider (e.g. "cursor-cli")
	ProviderName string
}

// CLIAgentOption is a functional option for CLIAgentProvider configuration
type CLIAgentOption func(*CLIAgentProvider)

// WithBaseArgs overrides the default base arguments
func WithBaseArgs(args []string) CLIAgentOption {
	return func(p *CLIAgentProvider) {
		p.BaseArgs = args
	}
}

// WithEnv adds an environment variable to the subprocess
func WithEnv(key, value string) CLIAgentOption {
	return func(p *CLIAgentProvider) {
		p.Env[key] = value
	}
}

// WithTimeout overrides the default timeout
func WithTimeout(d time.Duration) CLIAgentOption {
	return func(p *CLIAgentProvider) {
		p.Timeout = d
	}
}

// WithModel sets the display model name
func WithModel(model string) CLIAgentOption {
	return func(p *CLIAgentProvider) {
		p.Model = model
	}
}

// NewCLIAgentProvider creates a new CLI agent provider.
//
// command: the CLI binary name (e.g. "agent" for Cursor CLI)
// workDir: working directory for the subprocess (codebase root)
// providerName: display name for the provider (e.g. "cursor-cli")
// opts: functional options for additional configuration
func NewCLIAgentProvider(command, workDir, providerName string, opts ...CLIAgentOption) *CLIAgentProvider {
	p := &CLIAgentProvider{
		Command:      command,
		BaseArgs:     []string{"-p", "--output-format", "text"},
		WorkDir:      workDir,
		Timeout:      300 * time.Second,
		Env:          make(map[string]string),
		Model:        command + "-agent",
		ProviderName: providerName,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// NewCursorCLIProvider is a convenience constructor for Cursor CLI integration.
// It configures the provider with Cursor-specific defaults.
func NewCursorCLIProvider(workDir, apiKey string, opts ...CLIAgentOption) *CLIAgentProvider {
	p := NewCLIAgentProvider("agent", workDir, "cursor-cli", opts...)
	p.Model = "cursor-agent"

	if apiKey != "" {
		p.Env["CURSOR_API_KEY"] = apiKey
	}

	return p
}

// NewGeminiCLIProvider is a convenience constructor for Gemini CLI integration.
// It configures the provider with Gemini-specific defaults including --yolo
// to auto-approve tool use in headless mode.
func NewGeminiCLIProvider(workDir string, opts ...CLIAgentOption) *CLIAgentProvider {
	p := NewCLIAgentProvider("gemini", workDir, "gemini-cli", opts...)
	p.Model = "gemini-agent"
	// -p must be last: it takes the next arg as the prompt value (unlike Cursor's
	// boolean -p flag). GenerateResponse appends the prompt text after BaseArgs.
	// Tool approval is handled externally via the BeforeTool hook; no --yolo needed.
	p.BaseArgs = []string{"--output-format", "text", "-p"}

	return p
}

// IsCLIInstalled checks whether the configured CLI binary is available on PATH.
func (c *CLIAgentProvider) IsCLIInstalled() bool {
	_, err := exec.LookPath(c.Command)
	return err == nil
}

// GenerateResponse invokes the CLI agent with the given prompt and returns its response.
// Conversation history is injected into the prompt as context.
func (c *CLIAgentProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	// CLI agents don't use system role separation -- recombine into a single prompt
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	combinedPrompt := prompt
	if systemPrompt != "" {
		combinedPrompt = systemPrompt + "\n\n" + userMessage
	}

	// Build full prompt with conversation context
	fullPrompt := c.buildPromptWithHistory(combinedPrompt, conversationHistory)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// Build command arguments: base args + the prompt
	args := make([]string, len(c.BaseArgs))
	copy(args, c.BaseArgs)
	args = append(args, fullPrompt)

	log.Printf("[CLIAgent/%s] Invoking: %s %v (workDir: %s, timeout: %s)",
		c.ProviderName, c.Command, c.BaseArgs, c.WorkDir, c.Timeout)

	// Create command
	cmd := exec.CommandContext(timeoutCtx, c.Command, args...)

	// Set working directory
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}

	// Build environment: inherit current env + add custom vars
	cmd.Env = os.Environ()
	for key, value := range c.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()

	// Execute
	err := cmd.Run()
	duration := time.Since(start)

	log.Printf("[CLIAgent/%s] Completed in %s (exit: %v)", c.ProviderName, duration, err)

	if err != nil {
		// Check if it was a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%w after %s", ErrCLIProviderTimeout, c.Timeout)
		}

		// Keep raw stderr in logs but return a user-safe summarized error.
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			log.Printf("[CLIAgent/%s] stderr: %s", c.ProviderName, stderrStr)
			return "", fmt.Errorf("CLI agent failed: %s", truncateError(stderrStr))
		}
		return "", fmt.Errorf("CLI agent failed: %w", err)
	}

	// Parse the output
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		// Check stderr for any useful info
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			log.Printf("[CLIAgent/%s] empty stdout, stderr: %s", c.ProviderName, stderrStr)
			return "", fmt.Errorf("CLI agent returned empty output: %s", truncateError(stderrStr))
		}
		return "", fmt.Errorf("CLI agent returned empty output")
	}

	// Try to parse as JSON first (if --output-format json was used)
	parsed, err := c.parseJSONOutput(output)
	if err == nil && parsed != "" {
		return parsed, nil
	}

	// Otherwise return raw text output
	return output, nil
}

// GenerateVisionResponse is not supported by CLI agents.
// CLI agents handle images through file path references in the prompt.
func (c *CLIAgentProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	return "", fmt.Errorf("vision not directly supported by CLI agent provider (%s). "+
		"Include file paths in the prompt instead", c.ProviderName)
}

// GetModel returns the display model name
func (c *CLIAgentProvider) GetModel() string {
	return c.Model
}

// SupportsStreaming returns true -- CLI agents stream stdout in real time.
func (c *CLIAgentProvider) SupportsStreaming() bool { return true }

// GenerateResponseStream invokes the CLI agent and streams its stdout
// in chunks back through the returned channel. Uses a pipe for stdout
// and captures stderr separately so error dumps don't leak into chat.
func (c *CLIAgentProvider) GenerateResponseStream(ctx context.Context, prompt string, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	combinedPrompt := prompt
	if systemPrompt != "" {
		combinedPrompt = systemPrompt + "\n\n" + userMessage
	}
	fullPrompt := c.buildPromptWithHistory(combinedPrompt, conversationHistory)

	timeoutCtx, cancel := context.WithTimeout(ctx, c.Timeout)

	args := make([]string, len(c.BaseArgs))
	copy(args, c.BaseArgs)
	args = append(args, fullPrompt)

	cmd := exec.CommandContext(timeoutCtx, c.Command, args...)
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}
	cmd.Env = os.Environ()
	for key, value := range c.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start CLI agent: %w", err)
	}

	ch := make(chan StreamToken, 64)
	go func() {
		defer close(ch)
		defer cancel()

		buf := make([]byte, 4096)
		for {
			if timeoutCtx.Err() != nil {
				ch <- StreamToken{Error: fmt.Errorf("%w after %s", ErrCLIProviderTimeout, c.Timeout), Done: true}
				_ = cmd.Process.Kill()
				return
			}
			n, readErr := stdoutPipe.Read(buf)
			if n > 0 {
				clean := ansiRegex.ReplaceAllString(string(buf[:n]), "")
				if clean != "" {
					ch <- StreamToken{Content: clean}
				}
			}
			if readErr != nil {
				break
			}
		}

		if err := cmd.Wait(); err != nil {
			if timeoutCtx.Err() == context.DeadlineExceeded {
				ch <- StreamToken{Error: fmt.Errorf("%w after %s", ErrCLIProviderTimeout, c.Timeout), Done: true}
				return
			}
			stderrStr := strings.TrimSpace(stderrBuf.String())
			if stderrStr != "" {
				log.Printf("[CLIAgent/%s] stderr: %s", c.ProviderName, stderrStr)
				ch <- StreamToken{Error: fmt.Errorf("CLI agent error: %s", truncateError(stderrStr)), Done: true}
				return
			}
		}
		ch <- StreamToken{Done: true}
	}()

	return ch, nil
}

// truncateError extracts the first meaningful error line from verbose CLI output.
// Skips non-error preamble lines (e.g. "Loaded cached credentials.").
func truncateError(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip common non-error preamble from CLI tools
		lower := strings.ToLower(line)
		if strings.Contains(lower, "loaded") && strings.Contains(lower, "credentials") {
			continue
		}
		if len(line) > 200 {
			line = line[:200] + "..."
		}
		return line
	}
	// Fallback to first line if nothing matched
	first := strings.TrimSpace(lines[0])
	if len(first) > 200 {
		first = first[:200] + "..."
	}
	return first
}

// buildPromptWithHistory constructs a prompt that includes relevant conversation history
// so the CLI agent has context about the ongoing discussion.
func (c *CLIAgentProvider) buildPromptWithHistory(prompt string, history []protocol.Message) string {
	if len(history) == 0 {
		return prompt
	}

	var sb strings.Builder

	// Include limited history for context
	historyLimit := 8
	startIdx := 0
	if len(history) > historyLimit {
		startIdx = len(history) - historyLimit
	}

	sb.WriteString("Here is the recent conversation context:\n\n")
	for _, msg := range history[startIdx:] {
		role := msg.From.Name
		if role == "" {
			role = string(msg.From.Type)
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", role, msg.Content))
	}
	sb.WriteString("\n---\n\n")
	sb.WriteString("Now respond to the following:\n\n")
	sb.WriteString(prompt)

	return sb.String()
}

// cliJSONResponse represents the JSON output format from Cursor CLI (--output-format json).
// The structure captures the result field which contains the agent's final response.
type cliJSONResponse struct {
	Result string `json:"result"`
	// Cursor CLI may include additional fields
	Model      string `json:"model,omitempty"`
	DurationMS int    `json:"duration_ms,omitempty"`
}

// parseJSONOutput attempts to parse the CLI output as JSON and extract the response text.
func (c *CLIAgentProvider) parseJSONOutput(output string) (string, error) {
	var resp cliJSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return "", err
	}
	if resp.Result != "" {
		return resp.Result, nil
	}
	return "", fmt.Errorf("no result field in JSON output")
}
