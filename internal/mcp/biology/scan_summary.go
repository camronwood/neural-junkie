package biology

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/scansummary"
)

func summarizeScanSummaryPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	dir, err := scansummary.ResolveSummaryDir(path)
	if err != nil {
		return "", err
	}
	doc, err := scansummary.LoadMetadata(dir)
	if err != nil {
		return "", err
	}
	stats := scansummary.BuildSummaryStats(dir, doc)
	return scansummary.FormatSummaryMarkdown(stats), nil
}
