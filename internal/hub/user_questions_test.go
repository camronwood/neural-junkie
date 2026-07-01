package hub

import (
	"testing"
	"time"
)

func TestUserQuestionManager_AskAndAnswer(t *testing.T) {
	h := NewHub()
	uqm := NewUserQuestionManager(h)

	done := make(chan string, 1)
	go func() {
		answer, err := uqm.Ask("a1", "TestAgent", "general", "Which framework?", []string{"React", "Vue"}, 2*time.Second)
		if err != nil {
			t.Errorf("Ask: %v", err)
			done <- ""
			return
		}
		done <- answer
	}()

	time.Sleep(50 * time.Millisecond)
	pending := uqm.ListPending()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending question, got %d", len(pending))
	}
	if pending[0].Question != "Which framework?" {
		t.Fatalf("question=%q", pending[0].Question)
	}
	if err := uqm.Answer(pending[0].ID, "React"); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	answer := <-done
	if answer != "React" {
		t.Fatalf("answer=%q want React", answer)
	}
}
