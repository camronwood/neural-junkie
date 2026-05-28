package scananalysis

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/scansummary"
)

var scanExportSubdirs = []string{"scan-export", "scan_export", "scan", "summary"}

// ResolveLinkedScanDir finds a sibling scan summary directory for an analysis export.
func ResolveLinkedScanDir(analysisDir string) string {
	analysisDir = strings.TrimSpace(analysisDir)
	if analysisDir == "" {
		return ""
	}

	// analysisDir is the root containing reports/ — scan metadata at same level
	metaSame := filepath.Join(analysisDir, scansummary.MetadataFileName)
	if _, err := os.Stat(metaSame); err == nil {
		return analysisDir
	}

	// Common layout: analysisDir/scan-export/imageMetadata.json
	for _, sub := range scanExportSubdirs {
		cand := filepath.Join(analysisDir, sub)
		meta := filepath.Join(cand, scansummary.MetadataFileName)
		if _, err := os.Stat(meta); err == nil {
			if sub == "" {
				return analysisDir
			}
			return sub
		}
	}

	// analysisDir might be reports/ — scan at parent
	parent := filepath.Dir(analysisDir)
	if filepath.Base(analysisDir) == ReportsDirName {
		metaParent := filepath.Join(parent, scansummary.MetadataFileName)
		if _, err := os.Stat(metaParent); err == nil {
			return parent
		}
		for _, sub := range scanExportSubdirs {
			if sub == "" {
				continue
			}
			meta := filepath.Join(parent, sub, scansummary.MetadataFileName)
			if _, err := os.Stat(meta); err == nil {
				return filepath.Join(parent, sub)
			}
		}
	}

	return ""
}
