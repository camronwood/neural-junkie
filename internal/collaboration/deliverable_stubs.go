package collaboration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const deliverableStubMarker = "_Initial stub created when the plan was approved"

// IsDeliverableStubContent reports plan-approval placeholder file bodies.
func IsDeliverableStubContent(body []byte) bool {
	return strings.Contains(string(body), deliverableStubMarker)
}

// IsDeliverableStubFile reads up to 4KB from absPath and checks for the stub marker.
func IsDeliverableStubFile(absPath string) bool {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}
	if len(data) > 4096 {
		data = data[:4096]
	}
	return IsDeliverableStubContent(data)
}

// MaterializePlanDeliverableStubs creates placeholder markdown files for deliverable paths
// referenced in the approved plan/tasks under the collaboration working directory.
func MaterializePlanDeliverableStubs(c *Collaboration) ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("collaboration is required")
	}
	root := strings.TrimSpace(c.WorkingDirectory)
	if root == "" {
		root = PlannedOutputDirectory(c, "")
	}
	if root == "" {
		return nil, nil
	}
	paths := collectPlanDeliverablePaths(c)
	if len(paths) == 0 {
		return nil, nil
	}
	var created []string
	seen := make(map[string]struct{})
	for _, rel := range paths {
		rel = normalizeDeliverableRelPath(c, rel)
		if rel == "" {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return created, fmt.Errorf("mkdir deliverable parent: %w", err)
		}
		header := fmt.Sprintf("# %s\n\n%s. Replace with task output._\n", filepath.Base(rel), deliverableStubMarker)
		if err := os.WriteFile(abs, []byte(header), 0644); err != nil {
			return created, fmt.Errorf("write stub %s: %w", rel, err)
		}
		created = append(created, rel)
	}
	return created, nil
}

func collectPlanDeliverablePaths(c *Collaboration) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(rel string) {
		rel = strings.Trim(rel, "`\"' ")
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			return
		}
		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}
	if c.Plan != nil {
		for _, rel := range taskPathTokenRE.FindAllString(c.Plan.Content, -1) {
			add(sanitizePathToken(rel))
		}
	}
	for _, t := range c.Tasks {
		for _, rel := range ReferencedDeliverablePaths(t) {
			add(rel)
		}
	}
	return out
}

func normalizeDeliverableRelPath(c *Collaboration, rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return ""
	}
	if UsesProjectCollabDir(c) {
		prefix := ProjectCollabRelPath(c.ID) + "/"
		if strings.HasPrefix(rel, prefix) {
			return strings.TrimPrefix(rel, prefix)
		}
		shortID := c.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		altPrefix := ProjectCollabsDirName + "/" + shortID + "/"
		if strings.HasPrefix(rel, altPrefix) {
			return strings.TrimPrefix(rel, altPrefix)
		}
		return rel
	}
	prefix := ProjectCollabRelPath(c.ID) + "/"
	if strings.HasPrefix(rel, prefix) {
		return strings.TrimPrefix(rel, prefix)
	}
	shortID := c.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	altPrefix := ProjectCollabsDirName + "/" + shortID + "/"
	if strings.HasPrefix(rel, altPrefix) {
		return strings.TrimPrefix(rel, altPrefix)
	}
	return rel
}

// NormalizeDeliverableRelPathForRoot maps plan-relative deliverable paths to the execution workspace root.
func NormalizeDeliverableRelPathForRoot(c *Collaboration, rel string) string {
	return normalizeDeliverableRelPath(c, rel)
}
