package cad

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// Param describes one OpenSCAD customizer-style variable.
type Param struct {
	Name    string  `json:"name"`
	Value   string  `json:"value"`
	Section string  `json:"section,omitempty"`
	Comment string  `json:"comment,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Step    *float64 `json:"step,omitempty"`
}

var (
	paramAssignRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*([^;]+);`)
	sectionRe     = regexp.MustCompile(`/\*\s*\[([^\]]+)\]\s*\*/`)
	rangeRe     = regexp.MustCompile(`\[(-?\d+(?:\.\d+)?)\s*:\s*(-?\d+(?:\.\d+)?)(?:\s*:\s*(-?\d+(?:\.\d+)?))?\]`)
)

// ParseParams extracts top-level assignments and Customizer hints from OpenSCAD source.
func ParseParams(source string) []Param {
	var out []Param
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(source))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") && !strings.Contains(line, "[") {
			if m := sectionRe.FindStringSubmatch(line); len(m) == 2 {
				section = strings.TrimSpace(m[1])
			}
			continue
		}
		if m := sectionRe.FindStringSubmatch(line); len(m) == 2 {
			section = strings.TrimSpace(m[1])
			continue
		}
		m := paramAssignRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		name := m[1]
		value := strings.TrimSpace(m[2])
		comment := ""
		if idx := strings.Index(line, "//"); idx >= 0 {
			comment = strings.TrimSpace(line[idx+2:])
		}
		p := Param{Name: name, Value: value, Section: section, Comment: comment}
		if rm := rangeRe.FindStringSubmatch(comment); len(rm) >= 3 {
			if minV, err := strconv.ParseFloat(rm[1], 64); err == nil {
				p.Min = &minV
			}
			if maxV, err := strconv.ParseFloat(rm[2], 64); err == nil {
				p.Max = &maxV
			}
			if len(rm) >= 4 && rm[3] != "" {
				if stepV, err := strconv.ParseFloat(rm[3], 64); err == nil {
					p.Step = &stepV
				}
			}
		}
		out = append(out, p)
	}
	return out
}
