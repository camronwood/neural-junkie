package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationError describes a schema violation at a JSON path.
type ValidationError struct {
	Path    string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return e.Path + ": " + e.Message
}

// ObjectSpec is a minimal JSON-object schema (required keys + type object).
type ObjectSpec struct {
	Required []string
}

// ParseObject unmarshals JSON and validates required top-level keys exist and are non-empty strings when string-typed.
func ParseObject(raw []byte, spec ObjectSpec) (map[string]any, error) {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, &ValidationError{Message: fmt.Sprintf("invalid JSON: %v", err)}
	}
	for _, key := range spec.Required {
		v, ok := obj[key]
		if !ok {
			return nil, &ValidationError{Path: key, Message: "required field missing"}
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			return nil, &ValidationError{Path: key, Message: "required string is empty"}
		}
	}
	return obj, nil
}

// ParseInto unmarshals into T after object validation.
func ParseInto[T any](raw []byte, spec ObjectSpec) (T, error) {
	var zero T
	if _, err := ParseObject(raw, spec); err != nil {
		return zero, err
	}
	if err := json.Unmarshal(raw, &zero); err != nil {
		return zero, &ValidationError{Message: err.Error()}
	}
	return zero, nil
}

// FormatErrors joins validation errors for LLM retry prompts.
func FormatErrors(errs ...error) string {
	var parts []string
	for _, err := range errs {
		if err == nil {
			continue
		}
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "; ")
}
