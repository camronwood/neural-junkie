package scananalysis

// HeaderData holds analysis run identifiers from results.json header_data.
type HeaderData struct {
	AnalysisDate  string `json:"analysis_date"`
	AnalysisID    string `json:"analysis_id"`
	ScanResultsID string `json:"scan_results_id"`
	ScanDate      string `json:"scan_date"`
	ProductID     string `json:"product_id"`
}

// ExperimentData holds plate and algorithm metadata from experiment_data.
type ExperimentData struct {
	AnalysisPlateMapID         string             `json:"analysis_plate_map_id"`
	AnalysisName               string             `json:"analysis_name"`
	AnalysisPlateMapName       string             `json:"analysis_plate_map_name"`
	ProductName                string             `json:"product_name"`
	PlateBarcode               string             `json:"plate_barcode"`
	ScanName                   string             `json:"scan_name"`
	SignalCalculationAlgorithm string             `json:"signal_calculation_algorithm"`
	InitialConcentrations      map[string]float64 `json:"initial_concentrations"`
	DilutionFactor             float64            `json:"dilution_factor"`
}

// StandardRow is one standard curve QC row from standard_report_data.
type StandardRow struct {
	Analyte                              string             `json:"analyte"`
	WellLabel                            string             `json:"well_label"`
	Concentration                        float64            `json:"concentration"`
	Replicates                           map[string]float64 `json:"replicates"`
	MeanReplicateIntensity               *float64           `json:"mean_replicate_intensity"`
	MeanReplicateCalculatedConcentration *float64           `json:"mean_replicate_calculated_concentration"`
	PercentBias                          *float64           `json:"percent_bias"`
	WithinLimitsOfQuantificationV2       bool               `json:"within_limits_of_quantification_v2"`
	UpperPercentDifferenceV2             *float64           `json:"upper_percent_difference_v2"`
	LowerPercentDifferenceV2             *float64           `json:"lower_percent_difference_v2"`
}

// UnknownReplicate is one replicate measurement for an unknown well.
type UnknownReplicate struct {
	ReplicateIndex int      `json:"replicate_index"`
	Signal         float64  `json:"signal"`
	Concentration  *float64 `json:"concentration"`
}

// UnknownRow is one unknown well result from unknown_report_data.
type UnknownRow struct {
	Analyte                       string             `json:"analyte"`
	WellLabel                     string             `json:"well_label"`
	Replicates                    []UnknownReplicate `json:"replicates"`
	MeanReplicateConcentration    *float64           `json:"mean_replicate_concentration"`
	StdevOfReplicateConcentration *float64           `json:"stdev_of_replicate_concentration"`
	WithinLimitsOfQuantification  bool               `json:"within_limits_of_quantification"`
	ConcentrationUnit             string             `json:"concentration_unit"`
}

// ValidationRow is a flat per-well per-analyte result from validation_data.
type ValidationRow struct {
	Analyte                 string   `json:"analyte"`
	Signal                  float64  `json:"signal"`
	WellRow                 string   `json:"well_row"`
	WellColumn              int      `json:"well_column"`
	WellReplicateIndex      int      `json:"well_replicate_index"`
	WellType                string   `json:"well_type"`
	WellLabel               string   `json:"well_label"`
	CalculatedConcentration *float64 `json:"calculated_concentration"`
	KnownConcentration      *float64 `json:"known_concentration,omitempty"`
	SeriesIndex             int      `json:"series_index,omitempty"`
}

// SpotIntensityRow is per-spot signal data from spot_intensities.
type SpotIntensityRow struct {
	WellRow                  string  `json:"well_row"`
	WellColumn               int     `json:"well_column"`
	WellReplicateIndex       int     `json:"well_replicate_index"`
	WellType                 string  `json:"well_type"`
	WellLabel                string  `json:"well_label"`
	Analyte                  string  `json:"analyte"`
	Row                      int     `json:"row"`
	Column                   int     `json:"column"`
	Signal                   float64 `json:"signal"`
	Background               float64 `json:"background"`
	SignalIntensityAlgorithm string  `json:"signal_intensity_algorithm"`
}

// LimitsOfQuantification holds LLOQ/ULOQ/LOD for one analyte.
type LimitsOfQuantification struct {
	LLOQ               string `json:"LLOQ"`
	ULOQ               string `json:"ULOQ"`
	LODLabel           string `json:"LOD_label"`
	LOD                string `json:"LOD"`
	ConcentrationUnits string `json:"concentration_units"`
}

// FitParameters holds 5PL curve fit params for one analyte.
type FitParameters struct {
	A *float64 `json:"a,omitempty"`
	B *float64 `json:"b,omitempty"`
	C *float64 `json:"c,omitempty"`
	D *float64 `json:"d,omitempty"`
	G *float64 `json:"g,omitempty"`
}

// Document is the parsed results.json root.
type Document struct {
	Header          HeaderData                        `json:"header_data"`
	Experiment      ExperimentData                    `json:"experiment_data"`
	StandardReport  map[string][]StandardRow          `json:"standard_report_data"`
	UnknownReport   map[string][]UnknownRow           `json:"unknown_report_data"`
	Validation      []ValidationRow                   `json:"validation_data"`
	SpotIntensities []SpotIntensityRow                `json:"spot_intensities"`
	LimitsOfQuant   map[string]LimitsOfQuantification `json:"limits_of_quantification"`
	FitParameters   map[string]FitParameters          `json:"fit_parameters"`
}

// IndexedDocument adds lookup indexes built from a Document.
type IndexedDocument struct {
	Document
	Analytes           []string
	ByWellAnalyte      map[string]*ValidationRow // key: wellID|analyte
	ByWell             map[string][]ValidationRow
	SpotsByWellAnalyte map[string][]SpotIntensityRow
}
