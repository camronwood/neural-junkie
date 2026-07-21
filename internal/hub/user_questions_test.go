package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
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

func TestUserQuestionManager_HasPendingOnChannel(t *testing.T) {
	h := NewHub()
	uqm := h.GetUserQuestionManager()
	if uqm == nil {
		t.Fatal("expected hub user question manager")
	}

	done := make(chan struct{})
	go func() {
		_, _ = uqm.Ask("a1", "TestAgent", "general", "Pick engine?", nil, 2*time.Second)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	if !uqm.HasPendingOnChannel("general") {
		t.Fatal("expected pending on general")
	}
	if !h.HasPendingUserQuestion("general") {
		t.Fatal("hub HasPendingUserQuestion should be true")
	}
	if !h.ShouldDeferAgents("general") {
		t.Fatal("ShouldDeferAgents should be true while ask_user pending")
	}

	pending := uqm.ListPending()
	if len(pending) != 1 {
		t.Fatalf("pending=%d", len(pending))
	}
	if err := uqm.Answer(pending[0].ID, "Bevy"); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	<-done
	if uqm.HasPendingOnChannel("general") {
		t.Fatal("expected no pending after answer")
	}
	if h.ShouldDeferAgents("general") {
		t.Fatal("ShouldDeferAgents should clear after answer")
	}
}

func TestUserQuestionManager_DedupSimilarAnswer(t *testing.T) {
	h := NewHub()
	uqm := h.GetUserQuestionManager()

	done := make(chan string, 1)
	go func() {
		ans, err := uqm.Ask("a1", "FE", "general", "What is the target platform for your game?", nil, 2*time.Second)
		if err != nil {
			t.Errorf("Ask: %v", err)
		}
		done <- ans
	}()
	time.Sleep(40 * time.Millisecond)
	pending := uqm.ListPending()
	if len(pending) != 1 {
		t.Fatalf("pending=%d", len(pending))
	}
	if err := uqm.Answer(pending[0].ID, "Desktop"); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if got := <-done; got != "Desktop" {
		t.Fatalf("got %q", got)
	}

	// Rephrased platform question should reuse the answer without a new card.
	reuse, err := uqm.Ask("a1", "FE", "general", "What is the target platform for your game (e.g., desktop, web, mobile)?", nil, time.Second)
	if err != nil {
		t.Fatalf("reuse Ask: %v", err)
	}
	if reuse != "Desktop" {
		t.Fatalf("reuse=%q want Desktop", reuse)
	}
	if len(uqm.ListPending()) != 0 {
		t.Fatal("dedup must not create a new pending question")
	}
}

func TestQuestionsSimilar_platform(t *testing.T) {
	if !questionsSimilar(
		"What is the target platform for your game? (e.g., Web, Desktop, Mobile, Terminal/TUI)",
		"What is the target platform for your game (e.g., desktop, web, mobile)?",
	) {
		t.Fatal("expected platform questions to match")
	}
}

func TestUserQuestionManager_CoalescesConcurrentSimilarPendingQuestions(t *testing.T) {
	h := NewHub()
	uqm := h.GetUserQuestionManager()
	results := make(chan string, 2)
	errs := make(chan error, 2)
	for _, question := range []string{
		"What is the target platform for your game?",
		"What target platform should the game use (desktop, web, or mobile)?",
	} {
		go func(q string) {
			answer, err := uqm.Ask("a1", "FrontendEngineer", "rustgame", q, nil, 2*time.Second)
			results <- answer
			errs <- err
		}(question)
	}

	deadline := time.Now().Add(time.Second)
	for {
		uqm.mu.Lock()
		var pending []*UserQuestion
		for _, q := range uqm.questions {
			if q.Status == UserQuestionPending {
				pending = append(pending, q)
			}
		}
		waiterCount := 0
		if len(pending) == 1 {
			waiterCount = len(uqm.waiters[pending[0].ID])
		}
		uqm.mu.Unlock()
		if len(pending) == 1 && waiterCount == 2 {
			if err := uqm.Answer(pending[0].ID, "Desktop"); err != nil {
				t.Fatalf("Answer: %v", err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected one pending question with two waiters, pending=%d waiters=%d", len(pending), waiterCount)
		}
		time.Sleep(5 * time.Millisecond)
	}

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("Ask: %v", err)
		}
		if answer := <-results; answer != "Desktop" {
			t.Fatalf("answer=%q want Desktop", answer)
		}
	}
}

func TestUserQuestionManager_ExpireStaleReturnsTimeoutError(t *testing.T) {
	h := NewHub()
	uqm := h.GetUserQuestionManager()
	result := make(chan error, 1)
	go func() {
		_, err := uqm.Ask("a1", "FrontendEngineer", "rustgame", "Which genre?", nil, time.Minute)
		result <- err
	}()

	var questionID string
	deadline := time.Now().Add(time.Second)
	for questionID == "" {
		pending := uqm.ListPending()
		if len(pending) == 1 {
			questionID = pending[0].ID
			uqm.mu.Lock()
			uqm.questions[questionID].CreatedAt = time.Now().Add(-UserQuestionTTL - time.Second)
			uqm.mu.Unlock()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("question did not become pending")
		}
		time.Sleep(5 * time.Millisecond)
	}

	uqm.expireStale()
	select {
	case err := <-result:
		if err == nil || err.Error() != "timed out waiting for user response" {
			t.Fatalf("expected timeout error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Ask remained blocked after stale cleanup")
	}
	if uqm.HasPendingOnChannel("rustgame") {
		t.Fatal("expired question must no longer defer agents")
	}
}

func TestHubUpsertUserQuestionMessage_replacesPendingHistoryRow(t *testing.T) {
	h := NewHub()
	pending := protocol.NewMessage(
		protocol.MessageTypeUserQuestion,
		"general",
		protocol.AgentInfo{ID: "a1", Name: "FrontendEngineer"},
		"Which genre?",
	)
	pending.Metadata["question_id"] = "q1"
	pending.Metadata["status"] = "pending"
	h.messages["general"] = append(h.messages["general"], pending)

	answered := protocol.NewMessage(
		protocol.MessageTypeUserQuestion,
		"general",
		pending.From,
		"**Answer:** RTS",
	)
	answered.Metadata["question_id"] = "q1"
	answered.Metadata["status"] = "answered"
	answered.Metadata["answer"] = "RTS"

	if !h.upsertUserQuestionMessage(answered) {
		t.Fatal("expected pending history row to be updated")
	}
	msgs, err := h.GetMessages("general", 50)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	var cards []*protocol.Message
	for _, msg := range msgs {
		if msg.Type == protocol.MessageTypeUserQuestion && msg.Metadata["question_id"] == "q1" {
			cards = append(cards, msg)
		}
	}
	if len(cards) != 1 {
		t.Fatalf("expected one user-question row, got %d", len(cards))
	}
	if cards[0].Content != "**Answer:** RTS" || cards[0].Metadata["status"] != "answered" {
		t.Fatalf("unexpected updated card: %+v", cards[0])
	}
}

func TestPauseAgentsForUserQuestion_abortsPeersButNotAsker(t *testing.T) {
	h := NewHub()
	asker := agent.NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"ui"}, ai.NewMockProvider(), h)
	asker.Info.ID = "asker"
	peer := agent.NewAgent(protocol.AgentTypeRust, "RustExpert", []string{"rust"}, ai.NewMockProvider(), h)
	peer.Info.ID = "peer"
	h.commandHandler = &CommandHandler{
		runtimeAgents: map[string]*agent.Agent{
			asker.Info.ID: asker,
			peer.Info.ID:  peer,
		},
		cliAgents:        make(map[string]*agent.Agent),
		repoAgents:       make(map[string]*agent.RepoAgent),
		confluenceAgents: make(map[string]*agent.ConfluenceAgent),
	}

	askerCancelled := false
	peerCancelled := false
	agent.RegisterGenCancelForTest(asker, "rustgame", func() { askerCancelled = true })
	agent.RegisterGenCancelForTest(peer, "rustgame", func() { peerCancelled = true })

	h.pauseAgentsForUserQuestion("rustgame", asker.Info.ID)
	if askerCancelled {
		t.Fatal("asking agent must remain alive while waiting for the user")
	}
	if !peerCancelled {
		t.Fatal("peer generation should be cancelled while user question is pending")
	}
}
