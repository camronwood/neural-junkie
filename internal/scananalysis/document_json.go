package scananalysis

import "encoding/json"

func (u *UnknownReplicate) UnmarshalJSON(data []byte) error {
	type alias struct {
		ReplicateIndex int        `json:"replicate_index"`
		Signal         flexFloat  `json:"signal"`
		Concentration  *flexFloat `json:"concentration"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	u.ReplicateIndex = a.ReplicateIndex
	u.Signal = a.Signal.float64()
	u.Concentration = flexFloatPtrFrom(a.Concentration)
	return nil
}

func (v *ValidationRow) UnmarshalJSON(data []byte) error {
	type alias struct {
		Analyte                 string     `json:"analyte"`
		Signal                  flexFloat  `json:"signal"`
		WellRow                 string     `json:"well_row"`
		WellColumn              int        `json:"well_column"`
		WellReplicateIndex      int        `json:"well_replicate_index"`
		WellType                string     `json:"well_type"`
		WellLabel               string     `json:"well_label"`
		CalculatedConcentration *flexFloat `json:"calculated_concentration"`
		KnownConcentration      *flexFloat `json:"known_concentration"`
		SeriesIndex             int        `json:"series_index"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	v.Analyte = a.Analyte
	v.Signal = a.Signal.float64()
	v.WellRow = a.WellRow
	v.WellColumn = a.WellColumn
	v.WellReplicateIndex = a.WellReplicateIndex
	v.WellType = a.WellType
	v.WellLabel = a.WellLabel
	v.CalculatedConcentration = flexFloatPtrFrom(a.CalculatedConcentration)
	v.KnownConcentration = flexFloatPtrFrom(a.KnownConcentration)
	v.SeriesIndex = a.SeriesIndex
	return nil
}

func (s *SpotIntensityRow) UnmarshalJSON(data []byte) error {
	type alias struct {
		WellRow                  string    `json:"well_row"`
		WellColumn               int       `json:"well_column"`
		WellReplicateIndex       int       `json:"well_replicate_index"`
		WellType                 string    `json:"well_type"`
		WellLabel                string    `json:"well_label"`
		Analyte                  string    `json:"analyte"`
		Row                      int       `json:"row"`
		Column                   int       `json:"column"`
		Signal                   flexFloat `json:"signal"`
		Background               flexFloat `json:"background"`
		SignalIntensityAlgorithm string    `json:"signal_intensity_algorithm"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	s.WellRow = a.WellRow
	s.WellColumn = a.WellColumn
	s.WellReplicateIndex = a.WellReplicateIndex
	s.WellType = a.WellType
	s.WellLabel = a.WellLabel
	s.Analyte = a.Analyte
	s.Row = a.Row
	s.Column = a.Column
	s.Signal = a.Signal.float64()
	s.Background = a.Background.float64()
	s.SignalIntensityAlgorithm = a.SignalIntensityAlgorithm
	return nil
}

func (s *StandardRow) UnmarshalJSON(data []byte) error {
	type alias struct {
		Analyte                              string               `json:"analyte"`
		WellLabel                            string               `json:"well_label"`
		Concentration                        flexFloat            `json:"concentration"`
		Replicates                           map[string]flexFloat `json:"replicates"`
		MeanReplicateIntensity               *flexFloat           `json:"mean_replicate_intensity"`
		MeanReplicateCalculatedConcentration *flexFloat           `json:"mean_replicate_calculated_concentration"`
		PercentBias                          *flexFloat           `json:"percent_bias"`
		WithinLimitsOfQuantificationV2       bool                 `json:"within_limits_of_quantification_v2"`
		UpperPercentDifferenceV2             *flexFloat           `json:"upper_percent_difference_v2"`
		LowerPercentDifferenceV2             *flexFloat           `json:"lower_percent_difference_v2"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	s.Analyte = a.Analyte
	s.WellLabel = a.WellLabel
	s.Concentration = a.Concentration.float64()
	s.Replicates = flexFloatMapToFloat64(a.Replicates)
	s.MeanReplicateIntensity = flexFloatPtrFrom(a.MeanReplicateIntensity)
	s.MeanReplicateCalculatedConcentration = flexFloatPtrFrom(a.MeanReplicateCalculatedConcentration)
	s.PercentBias = flexFloatPtrFrom(a.PercentBias)
	s.WithinLimitsOfQuantificationV2 = a.WithinLimitsOfQuantificationV2
	s.UpperPercentDifferenceV2 = flexFloatPtrFrom(a.UpperPercentDifferenceV2)
	s.LowerPercentDifferenceV2 = flexFloatPtrFrom(a.LowerPercentDifferenceV2)
	return nil
}

func (e *ExperimentData) UnmarshalJSON(data []byte) error {
	type alias struct {
		AnalysisPlateMapID         string               `json:"analysis_plate_map_id"`
		AnalysisName               string               `json:"analysis_name"`
		AnalysisPlateMapName       string               `json:"analysis_plate_map_name"`
		ProductName                string               `json:"product_name"`
		PlateBarcode               string               `json:"plate_barcode"`
		ScanName                   string               `json:"scan_name"`
		SignalCalculationAlgorithm string               `json:"signal_calculation_algorithm"`
		InitialConcentrations      map[string]flexFloat `json:"initial_concentrations"`
		DilutionFactor             flexFloat            `json:"dilution_factor"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	e.AnalysisPlateMapID = a.AnalysisPlateMapID
	e.AnalysisName = a.AnalysisName
	e.AnalysisPlateMapName = a.AnalysisPlateMapName
	e.ProductName = a.ProductName
	e.PlateBarcode = a.PlateBarcode
	e.ScanName = a.ScanName
	e.SignalCalculationAlgorithm = a.SignalCalculationAlgorithm
	e.InitialConcentrations = flexFloatMapToFloat64(a.InitialConcentrations)
	e.DilutionFactor = a.DilutionFactor.float64()
	return nil
}

func (f *FitParameters) UnmarshalJSON(data []byte) error {
	type alias struct {
		A *flexFloat `json:"a,omitempty"`
		B *flexFloat `json:"b,omitempty"`
		C *flexFloat `json:"c,omitempty"`
		D *flexFloat `json:"d,omitempty"`
		G *flexFloat `json:"g,omitempty"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	f.A = flexFloatPtrFrom(a.A)
	f.B = flexFloatPtrFrom(a.B)
	f.C = flexFloatPtrFrom(a.C)
	f.D = flexFloatPtrFrom(a.D)
	f.G = flexFloatPtrFrom(a.G)
	return nil
}

func (u *UnknownRow) UnmarshalJSON(data []byte) error {
	type alias struct {
		Analyte                       string             `json:"analyte"`
		WellLabel                     string             `json:"well_label"`
		Replicates                    []UnknownReplicate `json:"replicates"`
		MeanReplicateConcentration    *flexFloat         `json:"mean_replicate_concentration"`
		StdevOfReplicateConcentration *flexFloat         `json:"stdev_of_replicate_concentration"`
		WithinLimitsOfQuantification  bool               `json:"within_limits_of_quantification"`
		ConcentrationUnit             string             `json:"concentration_unit"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	u.Analyte = a.Analyte
	u.WellLabel = a.WellLabel
	u.Replicates = a.Replicates
	u.MeanReplicateConcentration = flexFloatPtrFrom(a.MeanReplicateConcentration)
	u.StdevOfReplicateConcentration = flexFloatPtrFrom(a.StdevOfReplicateConcentration)
	u.WithinLimitsOfQuantification = a.WithinLimitsOfQuantification
	u.ConcentrationUnit = a.ConcentrationUnit
	return nil
}
