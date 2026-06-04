package scananalysis

// LLOQThresholdsPGML maps analyte names to maximum acceptable LLOQ (pg/mL) for Human Inflammatory 12-Plex.
// Ported from secondary-analysis-tools/12PlexQC_V1.py.
var LLOQThresholdsPGML = map[string]float64{
	"IL-1alpha": 0.02,
	"IL-1beta":  0.05,
	"IL-2":      0.05,
	"IL-4":      0.01,
	"IL-5":      0.01,
	"IL-6":      0.05,
	"IL-8":      0.05,
	"IL-10":     0.02,
	"IL-13":     0.05,
	"IFN-gamma": 0.05,
	"IL-12p70":  0.02,
	"TNF-alpha": 0.25,
}

const (
	ULOQMinimumPGML     = 200.0
	IntraplateCVMaxPct  = 25.0
	ColumnRowDevMaxPct  = 25.0
	SpikeRecoveryMinPct = 70.0
	SpikeRecoveryMaxPct = 130.0
	DefaultSpikePct     = 4.0
	DefaultSpikeDilution  = 4.0
)

// LLOQThresholdFor returns the LLOQ threshold for an analyte, or false if unknown.
func LLOQThresholdFor(analyte string) (float64, bool) {
	v, ok := LLOQThresholdsPGML[analyte]
	return v, ok
}
