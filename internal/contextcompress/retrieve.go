package contextcompress

import (
	"fmt"
	"regexp"
	"strings"
)

// Real refs are "ctx-" + 12 lowercase hex chars (see makeRef).
var validContextRef = regexp.MustCompile(`^ctx-[0-9a-f]{12}$`)

// placeholderContextRefs are schema/docs examples models sometimes copy literally.
var placeholderContextRefs = map[string]struct{}{
	"ctx-abc123":       {},
	"ctx-example":      {},
	"ctx-xxxxx":        {},
	"ctx-000000000000": {},
}

// ValidateContextRef returns an error if ref is empty, a known placeholder, or wrong shape.
func ValidateContextRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("ref is required — copy a ctx-… value from a compression marker in this turn (do not invent one)")
	}
	if _, bad := placeholderContextRefs[strings.ToLower(ref)]; bad {
		return fmt.Errorf("ref %q looks like a documentation example, not a real cache id — only call nj_retrieve_context with a ctx-… ref that appeared in a compressed tool result this turn", ref)
	}
	if !validContextRef.MatchString(ref) {
		return fmt.Errorf("ref %q is not a valid context cache id (expected ctx- plus 12 hex characters from a compression marker)", ref)
	}
	return nil
}

// Retrieve returns cached content, optionally filtered by query substring (line filter).
func Retrieve(store *Store, ref, query string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("context cache unavailable")
	}
	ref = strings.TrimSpace(ref)
	if err := ValidateContextRef(ref); err != nil {
		return "", err
	}
	data, ok := store.Get(ref)
	if !ok {
		return "", fmt.Errorf("ref %q not found or expired — it must come from a compression marker in this turn", ref)
	}
	text := string(data)
	q := strings.TrimSpace(query)
	if q == "" {
		return text, nil
	}
	qLower := strings.ToLower(q)
	var matched []string
	for _, ln := range strings.Split(text, "\n") {
		if strings.Contains(strings.ToLower(ln), qLower) {
			matched = append(matched, ln)
		}
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no lines matched query %q in ref %q", query, ref)
	}
	return strings.Join(matched, "\n"), nil
}
