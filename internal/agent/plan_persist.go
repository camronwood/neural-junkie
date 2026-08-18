package agent

import (
	"sync"

	"github.com/camronwood/neural-junkie/internal/plans"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	planStoreMu sync.Mutex
	planStore   = plans.Default()
)

func setPlanStoreForTest(s *plans.Store) func() {
	planStoreMu.Lock()
	prev := planStore
	planStore = s
	planStoreMu.Unlock()
	return func() {
		planStoreMu.Lock()
		planStore = prev
		planStoreMu.Unlock()
	}
}

func stampPersistedPlan(inbound *protocol.Message, responseMsg *protocol.Message, content string) {
	if inbound == nil || responseMsg == nil || !inbound.IdeEditorModeIsPlan() {
		return
	}
	planStoreMu.Lock()
	store := planStore
	planStoreMu.Unlock()
	if store == nil {
		return
	}
	rec, err := store.SaveFromMarkdown(content)
	if err != nil || rec == nil {
		return
	}
	if responseMsg.Metadata == nil {
		responseMsg.Metadata = make(map[string]interface{})
	}
	responseMsg.Metadata[protocol.MetaPlanID] = rec.ID
	responseMsg.Metadata[protocol.MetaPlanName] = rec.Name
}
