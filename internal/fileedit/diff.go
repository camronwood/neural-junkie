package fileedit

import (
	"fmt"
	"strings"
)

// UnifiedDiff generates a unified diff between old and new content for path.
func UnifiedDiff(path, oldContent, newContent string) string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	if len(oldLines) == 0 && len(newLines) == 0 {
		return ""
	}
	if strings.Join(oldLines, "\n") == strings.Join(newLines, "\n") {
		return ""
	}

	ops := diffOps(oldLines, newLines)
	if len(ops) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", path)
	fmt.Fprintf(&b, "+++ b/%s\n", path)

	const ctx = 3
	i := 0
	for i < len(ops) {
		for i < len(ops) && ops[i].tag == '=' {
			i++
		}
		if i >= len(ops) {
			break
		}
		start := i
		for i < len(ops) && ops[i].tag != '=' {
			i++
		}
		end := i

		hStart := start - ctx
		if hStart < 0 {
			hStart = 0
		}
		hEnd := end + ctx
		if hEnd > len(ops) {
			hEnd = len(ops)
		}

		oldStart, newStart := 1, 1
		if hStart < len(ops) {
			if ops[hStart].oldNum > 0 {
				oldStart = ops[hStart].oldNum
			}
			if ops[hStart].newNum > 0 {
				newStart = ops[hStart].newNum
			}
		}
		oldCount, newCount := 0, 0
		for _, op := range ops[hStart:hEnd] {
			switch op.tag {
			case '=':
				oldCount++
				newCount++
			case '-':
				oldCount++
			case '+':
				newCount++
			}
		}
		if oldCount == 0 {
			oldCount = 1
		}
		if newCount == 0 {
			newCount = 1
		}

		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for _, op := range ops[hStart:hEnd] {
			switch op.tag {
			case '=':
				b.WriteByte(' ')
				b.WriteString(op.oldLine)
			case '-':
				b.WriteByte('-')
				b.WriteString(op.oldLine)
			case '+':
				b.WriteByte('+')
				b.WriteString(op.newLine)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type diffOp struct {
	tag     byte // '=', '-', '+'
	oldLine string
	newLine string
	oldNum  int
	newNum  int
}

func diffOps(oldLines, newLines []string) []diffOp {
	n, m := len(oldLines), len(newLines)
	dp := lcsTable(oldLines, newLines)
	var ops []diffOp
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, diffOp{tag: '=', oldLine: oldLines[i-1], newLine: newLines[j-1], oldNum: i, newNum: j})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, diffOp{tag: '+', newLine: newLines[j-1], newNum: j})
			j--
		} else if i > 0 {
			ops = append(ops, diffOp{tag: '-', oldLine: oldLines[i-1], oldNum: i})
			i--
		}
	}
	reverseOps(ops)
	return ops
}

func reverseOps(ops []diffOp) {
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}
}

func lcsTable(a, b []string) [][]int {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp
}

// ChangedLineRange returns 1-based inclusive line range in old and new content that differ.
func ChangedLineRange(oldContent, newContent string) (oldFrom, oldTo, newFrom, newTo int) {
	ops := diffOps(splitLines(oldContent), splitLines(newContent))
	for _, op := range ops {
		switch op.tag {
		case '-':
			if oldFrom == 0 {
				oldFrom = op.oldNum
			}
			oldTo = op.oldNum
		case '+':
			if newFrom == 0 {
				newFrom = op.newNum
			}
			newTo = op.newNum
		}
	}
	return oldFrom, oldTo, newFrom, newTo
}
