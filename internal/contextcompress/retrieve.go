package contextcompress

import (
	"fmt"
	"strings"
)

// Retrieve returns cached content, optionally filtered by query substring (line filter).
func Retrieve(store *Store, ref, query string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("context cache unavailable")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("ref is required")
	}
	data, ok := store.Get(ref)
	if !ok {
		return "", fmt.Errorf("ref %q not found or expired", ref)
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
