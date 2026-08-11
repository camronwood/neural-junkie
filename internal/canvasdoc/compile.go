package canvasdoc

import (
	"regexp"
	"strings"
)

var (
	headingRE       = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	unorderedRE     = regexp.MustCompile(`^[-*+]\s+(.+?)\s*$`)
	orderedRE       = regexp.MustCompile(`^\d+\.\s+(.+?)\s*$`)
	imageRE         = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)$`)
	tableSepCellRE  = regexp.MustCompile(`^:?-{3,}:?$`)
	mermaidFenceRE  = regexp.MustCompile("(?is)```\\s*mermaid\\s*(?:\\r?\\n)?([\\s\\S]*?)```")
	codeFenceOpenRE = regexp.MustCompile("^```")
)

// CompileMarkdown lifts GFM constructs into hosted blocks.
func CompileMarkdown(src string) Document {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.TrimSpace(src)
	if src == "" {
		return Document{SchemaVersion: SchemaVersion, Blocks: []Block{}}
	}
	return Document{SchemaVersion: SchemaVersion, Blocks: compileSegments(src)}
}

func compileSegments(src string) []Block {
	var blocks []Block
	cursor := 0
	for _, loc := range mermaidFenceRE.FindAllStringSubmatchIndex(src, -1) {
		if loc[0] > cursor {
			blocks = append(blocks, compileProse(src[cursor:loc[0]])...)
		}
		source := strings.TrimSpace(src[loc[2]:loc[3]])
		if source != "" {
			blocks = append(blocks, Block{Type: TypeMermaid, Source: source})
		}
		cursor = loc[1]
	}
	if cursor < len(src) {
		blocks = append(blocks, compileProse(src[cursor:])...)
	}
	return blocks
}

func compileProse(src string) []Block {
	lines := strings.Split(strings.TrimSpace(src), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	var (
		blocks  []Block
		prose   []string
		items   []string
		ordered bool
		inList  bool
		inCode  bool
	)
	flushProse := func() {
		text := strings.TrimSpace(strings.Join(prose, "\n"))
		prose = prose[:0]
		if text != "" {
			blocks = append(blocks, Block{Type: TypeMarkdown, Source: text})
		}
	}
	flushList := func() {
		if !inList {
			return
		}
		blocks = append(blocks, Block{Type: TypeList, Ordered: ordered, Items: append([]string{}, items...)})
		items = items[:0]
		inList = false
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimSpace(line)

		if inCode {
			prose = append(prose, line)
			if codeFenceOpenRE.MatchString(trim) {
				inCode = false
			}
			continue
		}
		if codeFenceOpenRE.MatchString(trim) {
			flushList()
			inCode = true
			prose = append(prose, line)
			continue
		}

		if trim == "" {
			flushList()
			flushProse()
			continue
		}

		if m := headingRE.FindStringSubmatch(trim); m != nil {
			flushList()
			flushProse()
			level := len(m[1])
			if level > 3 {
				level = 3
			}
			blocks = append(blocks, Block{Type: TypeHeading, Level: level, Text: strings.TrimSpace(m[2])})
			continue
		}

		if m := imageRE.FindStringSubmatch(trim); m != nil {
			flushList()
			flushProse()
			blocks = append(blocks, Block{Type: TypeImage, Alt: m[1], Src: m[2]})
			continue
		}

		if isTableHeader(trim) && i+1 < len(lines) && isTableSeparator(strings.TrimSpace(lines[i+1])) {
			flushList()
			flushProse()
			table, consumed := parseTable(lines[i:])
			blocks = append(blocks, table)
			i += consumed - 1
			continue
		}

		if m := unorderedRE.FindStringSubmatch(trim); m != nil {
			flushProse()
			if inList && ordered {
				flushList()
			}
			inList = true
			ordered = false
			items = append(items, strings.TrimSpace(m[1]))
			continue
		}
		if m := orderedRE.FindStringSubmatch(trim); m != nil {
			flushProse()
			if inList && !ordered {
				flushList()
			}
			inList = true
			ordered = true
			items = append(items, strings.TrimSpace(m[1]))
			continue
		}

		flushList()
		prose = append(prose, line)
	}
	flushList()
	flushProse()
	return blocks
}

func isTableHeader(line string) bool {
	return strings.Count(line, "|") >= 2
}

func isTableSeparator(line string) bool {
	if strings.Count(line, "|") < 2 {
		return false
	}
	for _, cell := range splitTableRow(line) {
		if cell == "" {
			continue
		}
		if !tableSepCellRE.MatchString(cell) {
			return false
		}
	}
	return true
}

func parseTable(lines []string) (Block, int) {
	headers := splitTableRow(strings.TrimSpace(lines[0]))
	columns := make([]TableColumn, 0, len(headers))
	keys := make([]string, 0, len(headers))
	used := map[string]int{}
	for _, header := range headers {
		key := slugKey(header)
		if key == "" {
			key = "col"
		}
		if n := used[key]; n > 0 {
			key = key + "_" + itoa(n+1)
		}
		used[key]++
		label := header
		if label == "" {
			label = key
		}
		columns = append(columns, TableColumn{Key: key, Label: label})
		keys = append(keys, key)
	}
	rows := []map[string]string{}
	consumed := 2
	for i := 2; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if trim == "" || !strings.Contains(trim, "|") {
			break
		}
		if isTableSeparator(trim) {
			consumed++
			continue
		}
		cells := splitTableRow(trim)
		row := map[string]string{}
		for c, key := range keys {
			val := ""
			if c < len(cells) {
				val = cells[c]
			}
			row[key] = val
		}
		rows = append(rows, row)
		consumed++
	}
	return Block{Type: TypeTable, Columns: columns, Rows: rows}, consumed
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func slugKey(label string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(label)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('_')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [12]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
