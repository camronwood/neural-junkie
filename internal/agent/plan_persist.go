package agent

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/canvasdoc"
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

	// Create an nj.document artifact for the plan so the desktop can open it in
	// a canvas tab and show a Build button.
	artStore, err := getAgentArtifactStore()
	if err != nil {
		return
	}

	// Build the canvas document: overview callout prepended to the body blocks.
	// rec.Markdown is the full plan markdown; strip the YAML front-matter so
	// CompileMarkdown only sees the human-readable content.
	body := stripPlanFrontmatter(rec.Markdown)
	doc := canvasdoc.CompileMarkdown(body)
	if rec.Overview != "" {
		overviewBlock := canvasdoc.Block{
			Type:  canvasdoc.TypeCallout,
			Title: "Overview",
			Body:  rec.Overview,
		}
		doc.Blocks = append([]canvasdoc.Block{overviewBlock}, doc.Blocks...)
	}

	payload, _, err := canvasdoc.Marshal(doc)
	if err != nil {
		return
	}
	fallbackData, _ := json.Marshal(content)

	art := artifacts.Artifact{
		Kind:  "plan",
		Title: rec.Name,
		Renderer: artifacts.Renderer{
			ID:         canvasdoc.RendererID,
			APIVersion: "1",
			MediaType:  canvasdoc.MediaType,
		},
		Payload: payload,
		Fallback: &artifacts.Fallback{
			MediaType: "text/markdown",
			Data:      fallbackData,
		},
		Links: artifacts.ArtifactLinks{
			ChannelID: messageChannel(inbound),
		},
		Metadata: map[string]string{
			"plan_id": rec.ID,
		},
		Provenance: []artifacts.SourceReference{{
			Kind:  "plan",
			Label: rec.Name,
		}},
	}

	created, err := artStore.Create(art)
	if err != nil {
		return
	}

	responseMsg.SetArtifactReference(protocol.ArtifactReference{
		ID:         created.ID,
		Title:      created.Title,
		RendererID: canvasdoc.RendererID,
		MediaType:  canvasdoc.MediaType,
		Revision:   int64(created.Revision),
		Action:     "created",
	})
}

// stripPlanFrontmatter removes the YAML front-matter block from plan markdown so
// CompileMarkdown only sees the human-readable body.
func stripPlanFrontmatter(markdown string) string {
	s := strings.TrimSpace(markdown)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	rest := strings.TrimPrefix(s, "---")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return s
	}
	return strings.TrimSpace(rest[end+len("\n---"):])
}
