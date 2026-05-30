package export

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/learning"
)

const maxLearningRows = 50

// ExportLearningsRows converts confirmed learnings into Alpaca-style training rows.
func ExportLearningsRows(entries []learning.Entry) []Row {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) > maxLearningRows {
		entries = entries[:maxLearningRows]
	}
	out := make([]Row, 0, len(entries))
	for _, e := range entries {
		ctx := strings.TrimSpace(e.AgentName)
		if ctx == "" {
			ctx = e.AgentType
		}
		if e.Scope == learning.ScopeGlobal {
			ctx = "all experts"
		}
		if e.Scope == learning.ScopeCollaboration {
			ctx = "collaboration " + e.CollaborationID
		}
		out = append(out, Row{
			Instruction: "Apply this user-confirmed preference when relevant.",
			Input:       fmt.Sprintf("category=%s context=%s", e.Category, ctx),
			Output:      e.Content,
		})
	}
	return out
}

// MergeLearningsRows prepends learning rows before chat-derived rows.
func MergeLearningsRows(chat []Row, learnings []Row) []Row {
	if len(learnings) == 0 {
		return chat
	}
	out := make([]Row, 0, len(learnings)+len(chat))
	out = append(out, learnings...)
	out = append(out, chat...)
	return out
}
