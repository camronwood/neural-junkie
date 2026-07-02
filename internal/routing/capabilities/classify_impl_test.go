package capabilities

import "testing"

func TestClassifyImpl_firstAttemptBootFixUsesStandardImplement(t *testing.T) {
	main, tool := ClassifyImpl(ImplInput{
		TaskText:      "the app won't boot — make start-all fails",
		AgentType:     "frontend",
		BootFixIntent: true,
	})
	if main != TaskImplement || tool != TaskImplement {
		t.Fatalf("main=%q tool=%q, want implement/implement on first boot-fix attempt", main, tool)
	}
}

func TestClassifyImpl_repairTierUsesHeavy(t *testing.T) {
	main, tool := ClassifyImpl(ImplInput{
		TaskText:       "fix the compile error in src/App.tsx",
		AgentType:      "frontend",
		RepairAttempts: 1,
	})
	if main != TaskImplementHeavy || tool != TaskImplementHeavy {
		t.Fatalf("main=%q tool=%q, want implement_heavy after repair", main, tool)
	}
}

func TestClassifyImpl_verifyFailedUsesHeavy(t *testing.T) {
	main, tool := ClassifyImpl(ImplInput{
		TaskText:     "go test ./... still failing",
		AgentType:    "backend",
		VerifyFailed: true,
	})
	if main != TaskImplementHeavy || tool != TaskImplementHeavy {
		t.Fatalf("main=%q tool=%q, want implement_heavy when verify failed", main, tool)
	}
}
