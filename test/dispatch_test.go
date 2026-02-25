package test

import (
	"context"
	"testing"
	"time"

	"github.com/camron/ai-chat-room/internal/dispatch"
)

func TestCommandRegistry(t *testing.T) {
	registry := dispatch.NewCommandRegistry()

	t.Run("Read-only commands", func(t *testing.T) {
		testCases := []struct {
			plugin string
			subCmd string
			want   bool
		}{
			{"subenv", "list", true},
			{"subenv", "awake", true},
			{"sops", "view", true},
			{"plugin", "list", true},
		}

		for _, tc := range testCases {
			got := registry.IsReadOnly(tc.plugin, tc.subCmd)
			if got != tc.want {
				t.Errorf("IsReadOnly(%q, %q) = %v, want %v", tc.plugin, tc.subCmd, got, tc.want)
			}
		}
	})

	t.Run("Write commands require approval", func(t *testing.T) {
		testCases := []struct {
			plugin string
			subCmd string
			want   bool
		}{
			{"subenv", "create", false},
			{"subenv", "destroy", false},
			{"aws", "login", false},
			{"docker", "login", false},
			{"sops", "encrypt", false},
		}

		for _, tc := range testCases {
			got := registry.IsReadOnly(tc.plugin, tc.subCmd)
			if got != tc.want {
				t.Errorf("IsReadOnly(%q, %q) = %v, want %v (write commands should be false)", tc.plugin, tc.subCmd, got, tc.want)
			}
		}
	})

	t.Run("Unknown commands default to requiring approval", func(t *testing.T) {
		got := registry.IsReadOnly("unknown-plugin", "unknown-command")
		if got != false {
			t.Errorf("Unknown commands should default to requiring approval (false), got %v", got)
		}
	})

	t.Run("List plugins", func(t *testing.T) {
		plugins := registry.ListPlugins()
		if len(plugins) == 0 {
			t.Error("Expected some plugins to be registered")
		}

		// Check for known plugins
		hasSubenv := false
		hasAWS := false
		for _, p := range plugins {
			if p == "subenv" {
				hasSubenv = true
			}
			if p == "aws" {
				hasAWS = true
			}
		}

		if !hasSubenv {
			t.Error("Expected 'subenv' plugin to be registered")
		}
		if !hasAWS {
			t.Error("Expected 'aws' plugin to be registered")
		}
	})
}

func TestApprovalManager(t *testing.T) {
	executor := dispatch.NewExecutor()
	registry := dispatch.NewCommandRegistry()
	manager := dispatch.NewApprovalManager(executor, registry)
	defer manager.Stop()

	t.Run("Request and approve command", func(t *testing.T) {
		// Request approval
		approval, err := manager.RequestApproval(
			"user-123",
			"Test User",
			"general",
			"subenv",
			"create",
			[]string{"test-env"},
		)
		if err != nil {
			t.Fatalf("RequestApproval failed: %v", err)
		}

		if approval.ID == "" {
			t.Error("Expected approval ID to be set")
		}

		if approval.UserID != "user-123" {
			t.Errorf("Expected UserID to be 'user-123', got %q", approval.UserID)
		}

		// Get approval
		retrieved, err := manager.GetApproval(approval.ID)
		if err != nil {
			t.Fatalf("GetApproval failed: %v", err)
		}

		if retrieved.ID != approval.ID {
			t.Errorf("Expected ID %q, got %q", approval.ID, retrieved.ID)
		}

		// Approve command
		approved, err := manager.ApproveCommand(approval.ID, "user-123")
		if err != nil {
			t.Fatalf("ApproveCommand failed: %v", err)
		}

		if approved.ID != approval.ID {
			t.Errorf("Expected approved ID %q, got %q", approval.ID, approved.ID)
		}

		// Should be removed after approval
		_, err = manager.GetApproval(approval.ID)
		if err == nil {
			t.Error("Expected error getting approval after it was approved")
		}
	})

	t.Run("Only requestor can approve", func(t *testing.T) {
		approval, err := manager.RequestApproval(
			"user-123",
			"Test User",
			"general",
			"aws",
			"login",
			[]string{},
		)
		if err != nil {
			t.Fatalf("RequestApproval failed: %v", err)
		}

		// Try to approve with different user
		_, err = manager.ApproveCommand(approval.ID, "user-456")
		if err == nil {
			t.Error("Expected error when different user tries to approve")
		}
	})

	t.Run("Reject command", func(t *testing.T) {
		approval, err := manager.RequestApproval(
			"user-123",
			"Test User",
			"general",
			"docker",
			"login",
			[]string{},
		)
		if err != nil {
			t.Fatalf("RequestApproval failed: %v", err)
		}

		// Reject command
		rejected, err := manager.RejectCommand(approval.ID, "user-123")
		if err != nil {
			t.Fatalf("RejectCommand failed: %v", err)
		}

		if rejected.ID != approval.ID {
			t.Errorf("Expected rejected ID %q, got %q", approval.ID, rejected.ID)
		}

		// Should be removed after rejection
		_, err = manager.GetApproval(approval.ID)
		if err == nil {
			t.Error("Expected error getting approval after it was rejected")
		}
	})
}

func TestExecutor(t *testing.T) {
	executor := dispatch.NewExecutor()

	t.Run("Check if dispatch is installed", func(t *testing.T) {
		// This test just checks the method doesn't crash
		// Actual result depends on whether dispatch CLI is installed
		installed := executor.IsDispatchInstalled()
		t.Logf("Dispatch CLI installed: %v", installed)
	})

	t.Run("Execute command with timeout", func(t *testing.T) {
		if !executor.IsDispatchInstalled() {
			t.Skip("Dispatch CLI not installed, skipping execution test")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try to execute a simple read-only command
		result, err := executor.ExecuteCommand(ctx, "dispatch", "version", []string{})
		if err != nil {
			// Command may fail if dispatch is not configured, but shouldn't crash
			t.Logf("Command execution returned error (may be expected): %v", err)
		}

		if result == nil {
			t.Fatal("Expected result to be non-nil even on error")
		}

		t.Logf("Command: %s", result.Command)
		t.Logf("Exit code: %d", result.ExitCode)
		t.Logf("Duration: %v", result.Duration)
	})
}

func TestFormatter(t *testing.T) {
	formatter := dispatch.NewFormatter()

	t.Run("Format successful command", func(t *testing.T) {
		result := &dispatch.CommandResult{
			Command:  "dispatch subenv list",
			Plugin:   "subenv",
			SubCmd:   "list",
			Args:     []string{},
			ExitCode: 0,
			Stdout:   "test-env-1\ntest-env-2",
			Stderr:   "",
			Duration: 1200 * time.Millisecond,
			Success:  true,
		}

		output := formatter.FormatOutput(result)
		if output == "" {
			t.Error("Expected formatted output to be non-empty")
		}

		if !contains(output, "✅") {
			t.Error("Expected success emoji in output")
		}

		if !contains(output, "dispatch subenv list") {
			t.Error("Expected command string in output")
		}
	})

	t.Run("Format failed command", func(t *testing.T) {
		result := &dispatch.CommandResult{
			Command:  "dispatch subenv create test",
			Plugin:   "subenv",
			SubCmd:   "create",
			Args:     []string{"test"},
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "Error: permission denied",
			Duration: 500 * time.Millisecond,
			Success:  false,
		}

		output := formatter.FormatOutput(result)
		if output == "" {
			t.Error("Expected formatted output to be non-empty")
		}

		if !contains(output, "❌") {
			t.Error("Expected failure emoji in output")
		}

		if !contains(output, "permission denied") {
			t.Error("Expected error message in output")
		}
	})

	t.Run("Format approval request", func(t *testing.T) {
		approval := &dispatch.PendingApproval{
			ID:          "abc123",
			UserID:      "user-1",
			Username:    "Test User",
			Channel:     "general",
			Command:     "dispatch subenv create my-env",
			Plugin:      "subenv",
			SubCmd:      "create",
			Args:        []string{"my-env"},
			RequestedAt: time.Now(),
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}

		output := formatter.FormatApprovalRequest(approval)
		if output == "" {
			t.Error("Expected formatted approval request to be non-empty")
		}

		if !contains(output, "abc123") {
			t.Error("Expected approval ID in output")
		}

		if !contains(output, "/approve") {
			t.Error("Expected /approve command hint in output")
		}

		if !contains(output, "/reject") {
			t.Error("Expected /reject command hint in output")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
