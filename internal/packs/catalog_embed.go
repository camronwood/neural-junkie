package packs

import (
	_ "embed"
	"encoding/json"
)

//go:embed official_catalog.json
var officialCatalogJSON []byte

// builtinOfficialCatalog is the hub-shipped catalog snapshot (keep in sync with packs/catalog.json).
func builtinOfficialCatalog() (*Catalog, error) {
	var cat Catalog
	if err := json.Unmarshal(officialCatalogJSON, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

// mergeBuiltinOfficialPacks adds any official pack rows missing from the remote/cache catalog.
func mergeBuiltinOfficialPacks(cat *Catalog) (*Catalog, error) {
	builtin, err := builtinOfficialCatalog()
	if err != nil {
		return cat, err
	}
	if cat == nil {
		return orderCatalogPacks(builtin), nil
	}
	byID := make(map[string]CatalogEntry, len(cat.Packs))
	for _, e := range cat.Packs {
		id := e.ID
		if id == "" {
			continue
		}
		byID[id] = e
	}
	for _, e := range builtin.Packs {
		id := e.ID
		if id == "" {
			continue
		}
		if _, ok := byID[id]; !ok {
			byID[id] = e
		}
	}
	merged := &Catalog{Version: cat.Version}
	if merged.Version == 0 {
		merged.Version = builtin.Version
	}
	for _, e := range byID {
		merged.Packs = append(merged.Packs, e)
	}
	return orderCatalogPacks(merged), nil
}
