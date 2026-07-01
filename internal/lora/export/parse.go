package export

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const maxImportRows = 500

// ParseJSONL reads Alpaca-style JSONL training rows.
func ParseJSONL(r io.Reader, maxRows int) ([]Row, error) {
	if maxRows <= 0 {
		maxRows = maxImportRows
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []Row
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row Row
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if err := validateImportRow(row); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		row.SourceKind = "import"
		out = append(out, row)
		if len(out) >= maxRows {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid training rows found")
	}
	return out, nil
}

// ParsePastedRows accepts newline-delimited JSON objects or tab-separated instruction/output lines.
func ParsePastedRows(text string) ([]Row, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("paste is empty")
	}
	if strings.HasPrefix(text, "{") || strings.Contains(text, "\n{") {
		return ParseJSONL(strings.NewReader(text), maxImportRows)
	}
	var out []Row
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			return nil, fmt.Errorf("use JSON lines or tab-separated instruction<TAB>output")
		}
		row := Row{
			Instruction: strings.TrimSpace(parts[0]),
			Output:      strings.TrimSpace(strings.Join(parts[1:], "\t")),
			SourceKind:  "import",
		}
		if err := validateImportRow(row); err != nil {
			return nil, err
		}
		out = append(out, row)
		if len(out) >= maxImportRows {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid training rows found")
	}
	return out, nil
}

func validateImportRow(row Row) error {
	if strings.TrimSpace(row.Instruction) == "" {
		return fmt.Errorf("instruction is required")
	}
	if strings.TrimSpace(row.Output) == "" {
		return fmt.Errorf("output is required")
	}
	return nil
}
