package fileedit

import "fmt"

// Patch error codes.
const (
	ErrNotFound     = "not_found"
	ErrNotUnique    = "not_unique"
	ErrApplyFailed  = "apply_failed"
	ErrInvalidPatch = "invalid_patch"
	ErrOutOfScope   = "out_of_scope"
)

// PatchError describes a patch operation failure.
type PatchError struct {
	Code    string
	Message string
}

func (e *PatchError) Error() string {
	if e == nil {
		return "patch error"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
