package awsprofiles

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestValidateReadOnlyArgs(t *testing.T) {
	settings := config.AWSConfig{ReadOnly: boolPtr(true)}

	allowed := [][]string{
		{"sts", "get-caller-identity"},
		{"ec2", "describe-instances"},
		{"s3", "ls"},
		{"lambda", "list-functions"},
		{"iam", "list-roles"},
		{"cloudformation", "describe-stacks"},
	}
	for _, args := range allowed {
		if err := ValidateReadOnlyArgs(settings, args); err != nil {
			t.Errorf("expected allowed %v: %v", args, err)
		}
	}

	blocked := [][]string{
		{"ec2", "terminate-instances"},
		{"s3", "rm"},
		{"iam", "create-user"},
	}
	for _, args := range blocked {
		if err := ValidateReadOnlyArgs(settings, args); err == nil {
			t.Errorf("expected blocked %v", args)
		}
	}
}

func TestValidateReadOnlyDisabled(t *testing.T) {
	settings := config.AWSConfig{ReadOnly: boolPtr(false)}
	if err := ValidateReadOnlyArgs(settings, []string{"ec2", "terminate-instances"}); err != nil {
		t.Fatalf("read_only=false should allow any op: %v", err)
	}
}

func boolPtr(v bool) *bool { return &v }
