package collaboration

import "testing"

func TestScaledDiscussionConfig_threeAgents(t *testing.T) {
	cfg := ScaledDiscussionConfig(3)
	if cfg.MaxRounds != 3 {
		t.Fatalf("max rounds: got %d want 3", cfg.MaxRounds)
	}
	if cfg.TurnBudget != 2 {
		t.Fatalf("turn budget: got %d want 2", cfg.TurnBudget)
	}
	if cfg.MaxTotalMessages != 18 {
		t.Fatalf("max messages: got %d want 18", cfg.MaxTotalMessages)
	}
}

func TestDiscussionConfigWithScaledDefaults_respectsExplicitOverrides(t *testing.T) {
	cfg := DiscussionConfig{MaxRounds: 5}.WithScaledDefaults(3)
	if cfg.MaxRounds != 5 {
		t.Fatalf("explicit max rounds should be preserved, got %d", cfg.MaxRounds)
	}
	if cfg.MaxTotalMessages != 18 {
		t.Fatalf("scaled max messages for 3 agents: got %d", cfg.MaxTotalMessages)
	}
}
