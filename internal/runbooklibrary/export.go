package runbooklibrary

import "time"

// DefinitionBundleSchemaVersion is bumped whenever the shape of
// DefinitionBundle changes in a way that requires importer awareness.
const DefinitionBundleSchemaVersion = 1

// DefinitionBundle is the portable, shareable form of a RunbookDefinition:
// the definition itself plus enough envelope metadata for an importer to
// know where it came from and whether it understands the schema. This is
// the runbook-composition analog of the Share Agent MCP export bundle.
type DefinitionBundle struct {
	SchemaVersion int               `json:"schema_version"`
	ExportedAt    time.Time         `json:"exported_at"`
	Definition    RunbookDefinition `json:"definition"`
}

// NewDefinitionBundle wraps a definition for export/download.
func NewDefinitionBundle(def RunbookDefinition) DefinitionBundle {
	return DefinitionBundle{
		SchemaVersion: DefinitionBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Definition:    def,
	}
}

// ImportDefinitionBundle persists a definition bundle (or a bare
// RunbookDefinition, for backwards compatibility) as a new user definition.
// By default a fresh ID is minted so importing the same bundle twice, or
// importing a bundle exported from another installation, never collides
// with an existing local definition. Pass keepID=true to preserve the
// original ID (e.g. round-tripping within the same installation), in which
// case SaveUserDefinition will bump the version instead of colliding.
func ImportDefinitionBundle(def RunbookDefinition, keepID bool) (*RunbookDefinition, error) {
	if !keepID {
		def.ID = ""
	}
	def.Source = SourceUser
	def.PackID = ""
	// Let SaveUserDefinition assign the next version for this ID (fresh
	// definitions always start at v1 since the ID is new/blank).
	def.Version = 0
	return SaveUserDefinition(def)
}
