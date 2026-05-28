package biology

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/scananalysis"
)

func summarizeScanAnalysisPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	dir, err := scananalysis.ResolveAnalysisDir(path)
	if err != nil {
		return "", err
	}
	doc, source, err := scananalysis.LoadAnalysis(path)
	if err != nil {
		return "", err
	}
	stats := scananalysis.BuildAnalysisStats(dir, doc)
	stats.DataSource = string(source)
	stats.ProcessReportExcerpt = scananalysis.ProcessReportExcerpt(dir, 24)
	return scananalysis.FormatAnalysisMarkdown(stats), nil
}
