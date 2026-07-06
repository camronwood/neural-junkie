package runbooklibrary

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

var packRunbookIDSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// PackRunbookSource references a markdown runbook in an installed pack.
type PackRunbookSource struct {
	PackID string
	Path   string
	Title  string
}

// LoadPackDefinitions parses pack markdown runbooks into definitions.
func LoadPackDefinitions(sources []PackRunbookSource, readMarkdown func(packID, relPath string) (string, error)) ([]RunbookDefinition, error) {
	var out []RunbookDefinition
	for _, e := range sources {
		md, err := readMarkdown(e.PackID, e.Path)
		if err != nil {
			continue
		}
		tasks, err := collaboration.ParsePlanTasks(md, nil)
		if err != nil || len(tasks) == 0 {
			continue
		}
		id := PackRunbookDefinitionID(e.PackID, e.Path)
		title := e.Title
		if title == "" {
			title = id
		}
		out = append(out, RunbookDefinition{
			ID:          id,
			Version:     1,
			Title:       title,
			Description: "Pack runbook: " + e.Path,
			Source:      SourcePack,
			PackID:      e.PackID,
			Tasks:       tasks,
		})
	}
	return out, nil
}

// PackRunbookTemplateSource references a JSON runbook template in an installed pack.
type PackRunbookTemplateSource struct {
	PackID string
	Path   string
	Name   string
}

// LoadPackTemplateDefinitions parses pack JSON runbook templates into definitions.
func LoadPackTemplateDefinitions(sources []PackRunbookTemplateSource, readJSON func(packID, relPath string) ([]byte, error)) ([]RunbookDefinition, error) {
	var out []RunbookDefinition
	for _, e := range sources {
		raw, err := readJSON(e.PackID, e.Path)
		if err != nil {
			continue
		}
		var tpl collaboration.RunbookTemplate
		if err := json.Unmarshal(raw, &tpl); err != nil {
			continue
		}
		def := FromTemplate(tpl, SourcePack)
		def.PackID = e.PackID
		if def.ID == "" {
			def.ID = PackRunbookDefinitionID(e.PackID, e.Path)
		}
		out = append(out, def)
	}
	return out, nil
}

// PackRunbookDefinitionID builds a stable definition id for a pack markdown runbook.
func PackRunbookDefinitionID(packID, relPath string) string {
	base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	slug := packRunbookIDSanitizer.ReplaceAllString(strings.ToLower(base), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "runbook"
	}
	pack := packRunbookIDSanitizer.ReplaceAllString(strings.ToLower(packID), "-")
	pack = strings.Trim(pack, "-")
	if pack == "" {
		return slug
	}
	return pack + "-" + slug
}
