package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MessageSource reads hub message history for channel/repo exports.
type MessageSource interface {
	GetMessages(channel string, limit int) ([]*protocol.Message, error)
	GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error)
}

// Export writes Alpaca JSONL rows to outPath.
func Export(req Request, msgs MessageSource, collab *collaboration.Collaboration, outPath string) (int, error) {
	if req.MaxRows <= 0 {
		req.MaxRows = DefaultMaxRows
	}
	var rows []Row
	var err error
	switch req.Source {
	case SourceChannel:
		rows, err = exportChannel(req, msgs)
	case SourceCollaboration:
		rows, err = exportCollaboration(req, collab)
	case SourceRepo:
		rows, err = exportRepo(req, msgs)
	default:
		return 0, fmt.Errorf("unsupported source %q", req.Source)
	}
	if err != nil {
		return 0, err
	}
	if len(req.ExtraRows) > 0 {
		rows = MergeLearningsRows(rows, req.ExtraRows)
	}
	if len(rows) < MinRows {
		return len(rows), fmt.Errorf("only %d training rows (minimum %d)", len(rows), MinRows)
	}
	if err := writeJSONL(outPath, rows); err != nil {
		return 0, err
	}
	return len(rows), nil
}

// PreviewRowCount estimates export size without writing a file.
func PreviewRowCount(req Request, msgs MessageSource, collab *collaboration.Collaboration) (int, error) {
	if req.MaxRows <= 0 {
		req.MaxRows = DefaultMaxRows
	}
	switch req.Source {
	case SourceChannel:
		rows, err := exportChannel(req, msgs)
		return len(rows), err
	case SourceCollaboration:
		rows, err := exportCollaboration(req, collab)
		return len(rows), err
	case SourceRepo:
		rows, err := exportRepo(req, msgs)
		return len(rows), err
	default:
		return 0, fmt.Errorf("unsupported source %q", req.Source)
	}
}

func exportChannel(req Request, msgs MessageSource) ([]Row, error) {
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
	return pairsFromMessages(filterMessages(raw), req.MaxRows), nil
}

func exportRepo(req Request, msgs MessageSource) ([]Row, error) {
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
	return pairsFromMessages(filtered, req.MaxRows), nil
}

func exportCollaboration(req Request, collab *collaboration.Collaboration) ([]Row, error) {
	if collab == nil {
		return nil, fmt.Errorf("collaboration not found")
	}
	var rows []Row
	for _, t := range collab.Tasks {
		prompt := strings.TrimSpace(t.Description)
		if prompt == "" {
			prompt = strings.TrimSpace(t.Title)
		}
		out := strings.TrimSpace(t.Output)
		if prompt == "" || out == "" {
			continue
		}
		rows = append(rows, Row{
			Instruction: prompt,
			Output:      out,
		})
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

func pairsFromMessages(msgs []*protocol.Message, max int) []Row {
	var rows []Row
	var pendingUser string
	for _, m := range msgs {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if protocol.IsUserLikeSender(m.From) {
			pendingUser = content
			continue
		}
		if pendingUser == "" {
			continue
		}
		rows = append(rows, Row{
			Instruction: pendingUser,
			Output:      content,
		})
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
