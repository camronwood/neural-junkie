package awsprofiles

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// RunAWS runs an aws CLI invocation with profile and region from settings.
func RunAWS(ctx context.Context, settings config.AWSConfig, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("aws args required")
	}
	profile := settings.ProfileOrDefault()
	if profile == "" {
		return "", fmt.Errorf("aws profile not configured (Settings → Integrations)")
	}
	if !settings.ProfileAllowed(profile) {
		return "", fmt.Errorf("profile %q is not in allowed_profiles", profile)
	}
	if err := ValidateReadOnlyArgs(settings, args); err != nil {
		return "", err
	}
	full := append([]string{"--profile", profile, "--region", settings.DefaultRegionOrDefault(), "--output", "json"}, args...)
	cmd := exec.CommandContext(ctx, "aws", full...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return text, fmt.Errorf("%w: %s", err, text)
		}
		return "", err
	}
	return text, nil
}

// ValidateReadOnlyArgs rejects mutating aws subcommands when read_only is enabled.
func ValidateReadOnlyArgs(settings config.AWSConfig, args []string) error {
	if !settings.ReadOnlyEnabled() {
		return nil
	}
	if len(args) < 2 {
		return nil
	}
	service := strings.ToLower(strings.TrimSpace(args[0]))
	op := strings.ToLower(strings.TrimSpace(args[1]))
	switch service {
	case "sts":
		if op == "get-caller-identity" {
			return nil
		}
	case "s3":
		if op == "ls" || op == "list-buckets" || strings.HasPrefix(op, "api") && strings.Contains(op, "list") {
			return nil
		}
	case "ec2":
		if strings.HasPrefix(op, "describe-") {
			return nil
		}
	case "lambda":
		if strings.HasPrefix(op, "list-") || strings.HasPrefix(op, "get-") {
			return nil
		}
	case "cloudformation":
		if strings.HasPrefix(op, "describe-") || strings.HasPrefix(op, "list-") {
			return nil
		}
	case "iam":
		if strings.HasPrefix(op, "get-") || strings.HasPrefix(op, "list-") {
			return nil
		}
	}
	return fmt.Errorf("read-only mode: aws %s %s is not allowed", service, op)
}

// TestCallerIdentity verifies credentials for the configured profile.
func TestCallerIdentity(settings config.AWSConfig) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return RunAWS(ctx, settings, "sts", "get-caller-identity")
}
