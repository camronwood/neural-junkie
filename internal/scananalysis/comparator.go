package scananalysis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ComparatorArtifact is one file under a condition/plate folder.
type ComparatorArtifact struct {
	RelativePath string `json:"relative_path"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
}

// ComparatorArtifactGroup groups artifacts for one plate under a condition.
type ComparatorArtifactGroup struct {
	Condition string               `json:"condition"`
	Plate     string               `json:"plate"`
	Files     []ComparatorArtifact `json:"files"`
}

// ComparatorSummary holds parsed Summary Statistics from a Plate Comparator output folder.
type ComparatorSummary struct {
	AnalysisDir     string                         `json:"analysis_dir"`
	Conditions      []string                       `json:"conditions,omitempty"`
	SourcePlates    []string                       `json:"source_plates,omitempty"`
	LLOQULOQPath    string                         `json:"lloq_uloq_path,omitempty"`
	LLOQULOQRows    [][]string                     `json:"lloq_uloq_rows,omitempty"`
	PlateStats      map[string][][]string          `json:"plate_stats,omitempty"`
	InterplateStats map[string][][]string          `json:"interplate_stats,omitempty"`
	Artifacts       []ComparatorArtifactGroup      `json:"artifacts,omitempty"`
}

// IsComparatorAnalysisRoot reports whether dir contains Summary Statistics/LLOQs_and_ULOQs.csv.
func IsComparatorAnalysisRoot(dir string) bool {
	p := filepath.Join(dir, "Summary Statistics", "LLOQs_and_ULOQs.csv")
	_, err := os.Stat(p)
	return err == nil
}

// LoadComparatorSummary parses a Comparator Analysis output folder.
func LoadComparatorSummary(analysisDir string) (*ComparatorSummary, error) {
	if !IsComparatorAnalysisRoot(analysisDir) {
		return nil, fmt.Errorf("not a comparator analysis folder: %s", analysisDir)
	}
	sum := &ComparatorSummary{
		AnalysisDir:     analysisDir,
		PlateStats:      make(map[string][][]string),
		InterplateStats: make(map[string][][]string),
		LLOQULOQPath:    filepath.Join(analysisDir, "Summary Statistics", "LLOQs_and_ULOQs.csv"),
	}
	rows, err := readCSVFile(sum.LLOQULOQPath)
	if err != nil {
		return nil, err
	}
	sum.LLOQULOQRows = rows

	statsDir := filepath.Join(analysisDir, "Summary Statistics")
	entries, _ := os.ReadDir(statsDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "Plate ") || !strings.HasSuffix(e.Name(), " Summary Stats.csv") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "Plate "), " Summary Stats.csv")
		r, err := readCSVFile(filepath.Join(statsDir, e.Name()))
		if err == nil {
			sum.PlateStats[name] = r
		}
	}

	interplateDir := filepath.Join(analysisDir, "Interplate")
	if interEntries, err := os.ReadDir(interplateDir); err == nil {
		for _, e := range interEntries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".csv") {
				continue
			}
			r, err := readCSVFile(filepath.Join(interplateDir, e.Name()))
			if err == nil {
				sum.InterplateStats[e.Name()] = r
			}
		}
	}

	entries, _ = os.ReadDir(analysisDir)
	for _, e := range entries {
		if e.IsDir() && e.Name() != "Summary Statistics" && e.Name() != "Interplate" {
			sum.Conditions = append(sum.Conditions, e.Name())
			condPath := filepath.Join(analysisDir, e.Name())
			plates, _ := os.ReadDir(condPath)
			for _, p := range plates {
				if !p.IsDir() {
					continue
				}
				group := ComparatorArtifactGroup{
					Condition: e.Name(),
					Plate:     p.Name(),
				}
				walkArtifacts(filepath.Join(condPath, p.Name()), analysisDir, &group.Files)
				if len(group.Files) > 0 {
					sum.Artifacts = append(sum.Artifacts, group)
				}
				sum.SourcePlates = appendUnique(sum.SourcePlates, p.Name())
			}
		}
	}
	return sum, nil
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func walkArtifacts(dir, root string, out *[]ComparatorArtifact) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		kind := "file"
		if strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") {
			kind = "image"
		} else if strings.HasSuffix(lower, ".csv") {
			kind = "csv"
		}
		full := filepath.Join(dir, name)
		rel, err := filepath.Rel(root, full)
		if err != nil {
			rel = full
		}
		*out = append(*out, ComparatorArtifact{
			RelativePath: filepath.ToSlash(rel),
			Name:         name,
			Kind:         kind,
		})
	}
}

func readCSVFile(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return csv.NewReader(f).ReadAll()
}

// FormatComparatorMarkdown summarizes comparator output for MCP/chat.
func FormatComparatorMarkdown(sum *ComparatorSummary) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Comparator analysis: %s\n\n", filepath.Base(sum.AnalysisDir)))
	if len(sum.Conditions) > 0 {
		b.WriteString(fmt.Sprintf("- **Conditions:** %s\n", strings.Join(sum.Conditions, ", ")))
	}
	b.WriteString(fmt.Sprintf("- **Plates with summary stats:** %d\n", len(sum.PlateStats)))
	if len(sum.LLOQULOQRows) > 0 {
		b.WriteString(fmt.Sprintf("- **LLOQ/ULOQ table:** %d rows\n", len(sum.LLOQULOQRows)))
	}
	if len(sum.InterplateStats) > 0 {
		b.WriteString(fmt.Sprintf("- **Interplate CSV files:** %d\n", len(sum.InterplateStats)))
	}
	if len(sum.Artifacts) > 0 {
		n := 0
		for _, g := range sum.Artifacts {
			n += len(g.Files)
		}
		b.WriteString(fmt.Sprintf("- **Artifact files:** %d\n", n))
	}
	b.WriteString("\nOpen the folder in Neural Junkie to view heatmaps and interplate statistics.\n")
	return b.String()
}

// MarshalComparatorSummaryJSON returns API JSON for the comparator viewer.
func MarshalComparatorSummaryJSON(sum *ComparatorSummary) ([]byte, error) {
	return json.Marshal(sum)
}
