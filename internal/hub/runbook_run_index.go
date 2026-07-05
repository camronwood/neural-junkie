package hub

import (
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/runbookruns"
)

func (h *Hub) syncRunbookRunIndex(c *collaboration.Collaboration) {
	if c == nil || c.Source != collaboration.SourceRunbook || c.DefinitionID == "" {
		return
	}
	_ = runbookruns.AppendRun(runbookruns.RunRecord{
		ID:                c.ID,
		DefinitionID:      c.DefinitionID,
		DefinitionVersion: c.DefinitionVersion,
		RunNumber:         c.RunNumber,
		Phase:             string(c.Phase),
		Channel:           c.Channel,
		Title:             c.Title,
	})
}
