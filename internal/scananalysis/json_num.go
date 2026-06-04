package scananalysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// flexFloat accepts JSON numbers or numeric strings (some exports stringify signal values).
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*f = flexFloat(math.NaN())
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" || strings.EqualFold(s, "NaN") || strings.EqualFold(s, "null") {
			*f = flexFloat(math.NaN())
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("parse float string %q: %w", s, err)
		}
		*f = flexFloat(v)
		return nil
	}
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

func (f flexFloat) float64() float64 {
	return float64(f)
}

func flexFloatPtrFrom(f *flexFloat) *float64 {
	if f == nil {
		return nil
	}
	v := f.float64()
	return &v
}

func flexFloatMapToFloat64(m map[string]flexFloat) map[string]float64 {
	if m == nil {
		return nil
	}
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v.float64()
	}
	return out
}
