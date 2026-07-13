package hub

import (
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/runbookruns"
	"github.com/camronwood/neural-junkie/internal/workflow"
)

func (h *Hub) syncRunbookRunIndex(c *collaboration.Collaboration) {
	if c == nil || c.Source != collaboration.SourceRunbook || c.DefinitionID == "" {
		return
	}
	eventPath, _ := workflow.EventLogPath(c.ID)
	outcome := string(c.Phase)
	if c.Phase == collaboration.PhaseCompleted {
		outcome = "completed"
	} else if c.Phase == collaboration.PhaseCancelled {
		outcome = "cancelled"
	}
	_ = runbookruns.AppendRun(runbookruns.RunRecord{
		ID:                c.ID,
		DefinitionID:      c.DefinitionID,
		DefinitionVersion: c.DefinitionVersion,
		RunNumber:         c.RunNumber,
		Phase:             string(c.Phase),
		Channel:           c.Channel,
		Title:             c.Title,
		EventLogPath:      eventPath,
		Outcome:           outcome,
	})
}
