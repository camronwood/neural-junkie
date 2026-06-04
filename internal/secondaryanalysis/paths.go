package secondaryanalysis

import (
	"path/filepath"
	"strings"
)

// ResolveCumulativeQCDir returns the cumulative QC directory for a workspace.
// When settingsOverride is non-empty it wins; otherwise workspace/.neural-junkie/cumulative-qc.
func ResolveCumulativeQCDir(workspaceRoot, settingsOverride string) string {
	if d := strings.TrimSpace(settingsOverride); d != "" {
		return d
	}
	return filepath.Join(workspaceRoot, ".neural-junkie", "cumulative-qc")
}

// ResolveCumulativeSPCDir returns the cumulative SPC directory for a workspace.
func ResolveCumulativeSPCDir(workspaceRoot, settingsOverride string) string {
	if d := strings.TrimSpace(settingsOverride); d != "" {
		return filepath.Join(d, "spc")
	}
	return filepath.Join(workspaceRoot, ".neural-junkie", "cumulative-spc")
}

// AnalysisRunsDir is the default parent for job outputs inside a workspace.
func AnalysisRunsDir(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".neural-junkie", "analysis-runs")
}

// IsComparatorAnalysisDir reports whether dir looks like a Plate Comparator output folder.
func IsComparatorAnalysisDir(dirName string) bool {
	return strings.HasPrefix(dirName, "Comparator Analysis")
}
