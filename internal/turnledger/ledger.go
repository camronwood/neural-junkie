// Package turnledger persists an append-only per-channel turn ledger for long-conversation tracking.
package turnledger

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

const MaxExcerptRunes = 240

// Entry is one append-only turn in a channel ledger.
type Entry struct {
	TS          time.Time `json:"ts"`
	Channel     string    `json:"channel"`
	MessageID   string    `json:"message_id,omitempty"`
	Speaker     string    `json:"speaker"`
	SpeakerType string    `json:"speaker_type"` // human | agent
	MsgType     string    `json:"msg_type,omitempty"`
	Intent      string    `json:"intent,omitempty"`
	Mode        string    `json:"conversation_mode,omitempty"`
	Excerpt     string    `json:"excerpt,omitempty"`
	Entities    []string  `json:"entities,omitempty"`
	GoalID      string    `json:"goal_id,omitempty"`
	CollabID    string    `json:"collab_id,omitempty"`
	TraceID     string    `json:"trace_id,omitempty"`
}

var writeMu sync.Mutex

// testDir overrides ~/.neural-junkie/turn-ledgers when set (tests only).
var testDir string

var (
	backtickRE = regexp.MustCompile("`([^`\\n]{2,64})`")
	camelRE    = regexp.MustCompile(`\b([A-Z][a-z0-9]+(?:[A-Z][a-z0-9]+)+)\b`)
)

// SetDirForTest overrides the ledger directory (tests only). Pass "" to clear.
func SetDirForTest(dir string) {
	testDir = dir
}

func ledgerDir() (string, error) {
	if testDir != "" {
		return testDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "turn-ledgers"), nil
}

// SafeChannelFile maps a channel name to a filesystem-safe stem.
func SafeChannelFile(channel string) string {
	ch := strings.TrimSpace(channel)
	if ch == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range ch {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}

// Path returns the JSONL path for a channel.
func Path(channel string) (string, error) {
	dir, err := ledgerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, SafeChannelFile(channel)+".jsonl"), nil
}

// TruncateExcerpt bounds excerpt length for ledger storage / prompt overlay.
func TruncateExcerpt(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= MaxExcerptRunes {
		return s
	}
	return string(runes[:MaxExcerptRunes-1]) + "…"
}

// ExtractEntities pulls lightweight topic tokens (backticks + CamelCase).
func ExtractEntities(content string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(tok string) {
		tok = strings.TrimSpace(tok)
		if len(tok) < 2 || len(tok) > 64 {
			return
		}
		key := strings.ToLower(tok)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, tok)
		if len(out) >= 8 {
			return
		}
	}
	for _, m := range backtickRE.FindAllStringSubmatch(content, 8) {
		if len(m) > 1 {
			add(m[1])
		}
		if len(out) >= 8 {
			return out
		}
	}
	for _, m := range camelRE.FindAllStringSubmatch(content, 8) {
		if len(m) > 1 {
			add(m[1])
		}
		if len(out) >= 8 {
			return out
		}
	}
	return out
}

// Append writes one entry to turn-ledgers/{safe_channel}.jsonl.
func Append(channel string, ev Entry) error {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return nil
	}
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	ev.Channel = channel
	ev.Excerpt = TruncateExcerpt(ev.Excerpt)
	if len(ev.Entities) == 0 && ev.Excerpt != "" {
		ev.Entities = ExtractEntities(ev.Excerpt)
	}
	dir, err := ledgerDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	f, err := os.OpenFile(filepath.Join(dir, SafeChannelFile(channel)+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// ReadTail loads the last limit entries for a channel (oldest→newest within the tail).
func ReadTail(channel string, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	path, err := Path(channel)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev Entry
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		all = append(all, ev)
	}
	if err := scanner.Err(); err != nil {
		return all, err
	}
	if len(all) <= limit {
		return all, nil
	}
	return all[len(all)-limit:], nil
}

// FormatOverlay builds a compact prompt block from ledger rows.
func FormatOverlay(entries []Entry, maxRows int) string {
	if maxRows <= 0 {
		maxRows = 12
	}
	if len(entries) == 0 {
		return ""
	}
	if len(entries) > maxRows {
		entries = entries[len(entries)-maxRows:]
	}
	var b strings.Builder
	b.WriteString("=== TURN LEDGER (recent) ===\n")
	b.WriteString("Structured recent turns (speaker-attributed). Recent exchanges below remain ground truth.\n")
	for _, e := range entries {
		who := strings.TrimSpace(e.Speaker)
		if who == "" {
			who = "?"
		}
		kind := strings.TrimSpace(e.SpeakerType)
		if kind != "" {
			who = who + "/" + kind
		}
		line := TruncateExcerpt(e.Excerpt)
		if line == "" {
			line = "(" + strings.TrimSpace(e.MsgType) + ")"
		}
		b.WriteString("- ")
		b.WriteString(who)
		b.WriteString(": ")
		b.WriteString(line)
		if len(e.Entities) > 0 {
			b.WriteString(" [")
			b.WriteString(strings.Join(e.Entities, ", "))
			b.WriteString("]")
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}
