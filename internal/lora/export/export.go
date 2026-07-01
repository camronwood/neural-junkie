package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MessageSource reads hub message history for channel/repo exports.
type MessageSource interface {
	GetMessages(channel string, limit int) ([]*protocol.Message, error)
	GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error)
}

// CollectRows builds preview rows from the request without writing a file.
func CollectRows(req Request, msgs MessageSource, collab *collaboration.Collaboration) ([]PreviewRow, error) {
	if req.MaxRows <= 0 {
		req.MaxRows = DefaultMaxRows
	}
	var rows []PreviewRow
	var err error
	switch req.Source {
	case SourceChannel:
		rows, err = collectChannel(req, msgs)
	case SourceCollaboration:
		rows, err = collectCollaboration(req, collab)
	case SourceRepo:
		rows, err = collectRepo(req, msgs)
	default:
		return nil, fmt.Errorf("unsupported source %q", req.Source)
	}
	if err != nil {
		return nil, err
	}
	if !req.SinceTimestamp.IsZero() {
		rows = FilterSinceTimestamp(rows, req.SinceTimestamp)
	}
	if !req.SinceJobExportedAt.IsZero() {
		rows = FilterSinceTimestamp(rows, req.SinceJobExportedAt)
	}
	rows = mergeExtraRows(rows, req.ExtraRows)
	rows = applyRowFilters(rows, req)
	if req.MaxRows > 0 && len(rows) > req.MaxRows {
		rows = rows[:req.MaxRows]
	}
	return rows, nil
}

// Export writes Alpaca JSONL rows to outPath.
func Export(req Request, msgs MessageSource, collab *collaboration.Collaboration, outPath string) (int, error) {
	rows, err := CollectRows(req, msgs, collab)
	if err != nil {
		return 0, err
	}
	alpaca := previewToAlpaca(rows)
	if err := validateMinRows(len(alpaca)); err != nil {
		return len(alpaca), err
	}
	if err := writeJSONL(outPath, alpaca); err != nil {
		return 0, err
	}
	return len(alpaca), nil
}

// PreviewRowCount estimates export size without writing a file.
func PreviewRowCount(req Request, msgs MessageSource, collab *collaboration.Collaboration) (int, error) {
	rows, err := CollectRows(req, msgs, collab)
	if err != nil {
		return 0, err
	}
	return len(previewToAlpaca(rows)), nil
}

// PreviewRows returns rows with metadata for curation.
func PreviewRows(req Request, msgs MessageSource, collab *collaboration.Collaboration) ([]PreviewRow, error) {
	return CollectRows(req, msgs, collab)
}

func previewToAlpaca(rows []PreviewRow) []Row {
	out := make([]Row, 0, len(rows))
	curated := false
	for _, r := range rows {
		if r.Included {
			curated = true
			break
		}
	}
	for _, r := range rows {
		if curated && !r.Included {
			continue
		}
		out = append(out, Row{
			Instruction: r.Instruction,
			Input:       r.Input,
			Output:      r.Output,
		})
	}
	return out
}

func collectChannel(req Request, msgs MessageSource) ([]PreviewRow, error) {
	channel := strings.TrimSpace(req.SourceID)
	if channel == "" {
		return nil, fmt.Errorf("channel id is required")
	}
	var raw []*protocol.Message
	var err error
	if tid := strings.TrimSpace(req.ThreadID); tid != "" {
		raw, err = msgs.GetThreadMessages(tid, req.MaxRows*2)
	} else {
		raw, err = msgs.GetMessages(channel, req.MaxRows*2)
	}
	if err != nil {
		return nil, err
	}
	return pairsFromMessages(filterMessages(raw), req.MaxRows, string(SourceChannel), channel), nil
}

func collectRepo(req Request, msgs MessageSource) ([]PreviewRow, error) {
	channel := strings.TrimSpace(req.SourceID)
	agentName := strings.TrimSpace(req.AgentName)
	if channel == "" {
		return nil, fmt.Errorf("channel id is required for repo export")
	}
	raw, err := msgs.GetMessages(channel, req.MaxRows*3)
	if err != nil {
		return nil, err
	}
	filtered := filterMessages(raw)
	if agentName != "" {
		filtered = filterByAgentName(filtered, agentName)
	}
	return pairsFromMessages(filtered, req.MaxRows, string(SourceRepo), channel), nil
}

func collectCollaboration(req Request, collab *collaboration.Collaboration) ([]PreviewRow, error) {
	if collab == nil {
		return nil, fmt.Errorf("collaboration not found")
	}
	var rows []PreviewRow
	for _, t := range collab.Tasks {
		if req.ApprovedTasksOnly {
			if t.Status != collaboration.TaskCompleted {
				continue
			}
		}
		prompt := strings.TrimSpace(t.Description)
		if prompt == "" {
			prompt = strings.TrimSpace(t.Title)
		}
		out := strings.TrimSpace(t.Output)
		if prompt == "" || out == "" {
			continue
		}
		ref := strings.TrimSpace(t.ID)
		if ref == "" {
			ref = strings.TrimSpace(t.Title)
		}
		rows = append(rows, PreviewRow{Row: Row{
			RowID:       rowID(string(SourceCollaboration), collab.ID, ref),
			Instruction: prompt,
			Output:      out,
			SourceKind:  string(SourceCollaboration),
			SourceRef:   ref,
			Included:    true,
		}})
		if len(rows) >= req.MaxRows {
			break
		}
	}
	return rows, nil
}

func filterMessages(msgs []*protocol.Message) []*protocol.Message {
	var out []*protocol.Message
	for _, m := range msgs {
		if chatcontext.OmitFromLLMHistory(m) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func filterByAgentName(msgs []*protocol.Message, agentName string) []*protocol.Message {
	name := strings.ToLower(strings.TrimSpace(agentName))
	var out []*protocol.Message
	for _, m := range msgs {
		from := strings.ToLower(strings.TrimSpace(m.From.Name))
		if protocol.IsUserLikeSender(m.From) || from == name {
			out = append(out, m)
		}
	}
	return out
}

func pairsFromMessages(msgs []*protocol.Message, max int, sourceKind, sourceRef string) []PreviewRow {
	var rows []PreviewRow
	var pendingUser string
	var pendingAt time.Time
	var pendingID string
	for _, m := range msgs {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if protocol.IsUserLikeSender(m.From) {
			pendingUser = content
			pendingAt = m.Timestamp
			pendingID = m.ID
			continue
		}
		if pendingUser == "" {
			continue
		}
		msgAt := ""
		if !pendingAt.IsZero() {
			msgAt = pendingAt.UTC().Format(time.RFC3339)
		}
		rid := rowID(sourceKind, sourceRef, pendingID, m.ID)
		rows = append(rows, PreviewRow{Row: Row{
			RowID:       rid,
			Instruction: pendingUser,
			Output:      content,
			SourceKind:  sourceKind,
			SourceRef:   sourceRef,
			MessageAt:   msgAt,
			Included:    true,
		}})
		pendingUser = ""
		if len(rows) >= max {
			break
		}
	}
	return rows
}

func writeJSONL(path string, rows []Row) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return nil
}
