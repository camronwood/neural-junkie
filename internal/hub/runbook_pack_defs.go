package hub

import (
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func (h *Hub) packRunbookDefinitions() []runbooklibrary.RunbookDefinition {
	if h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return nil
	}
	cfg := h.commandHandler.appConfig
	statuses := cfg.ListPackStatus().Packs
	dirs := map[string]string{}
	manifests := map[string]*packs.Manifest{}
	for _, st := range statuses {
		if !st.Enabled {
			continue
		}
		dir, err := packs.InstalledPackDir(st.ID)
		if err != nil {
			continue
		}
		m, err := cfg.InstalledPackManifestByID(st.ID)
		if err != nil || m == nil {
			continue
		}
		dirs[st.ID] = dir
		manifests[st.ID] = m
	}
	entries, err := packs.ListPackRunbooks(dirs, manifests)
	if err != nil {
		return nil
	}
	sources := make([]runbooklibrary.PackRunbookSource, len(entries))
	for i, e := range entries {
		sources[i] = runbooklibrary.PackRunbookSource{PackID: e.PackID, Path: e.Path, Title: e.Title}
	}
	readFn := func(packID, relPath string) (string, error) {
		return packs.ReadPackRunbookMarkdown(dirs[packID], relPath)
	}
	defs, err := runbooklibrary.LoadPackDefinitions(sources, readFn)
	if err != nil {
		return nil
	}
	tplEntries, err := packs.ListPackRunbookTemplates(dirs, manifests)
	if err == nil && len(tplEntries) > 0 {
		tplSources := make([]runbooklibrary.PackRunbookTemplateSource, len(tplEntries))
		for i, e := range tplEntries {
			tplSources[i] = runbooklibrary.PackRunbookTemplateSource{PackID: e.PackID, Path: e.Path, Name: e.Name}
		}
		readJSON := func(packID, relPath string) ([]byte, error) {
			return packs.ReadPackRunbookTemplateJSON(dirs[packID], relPath)
		}
		tplDefs, err := runbooklibrary.LoadPackTemplateDefinitions(tplSources, readJSON)
		if err == nil {
			defs = append(defs, tplDefs...)
		}
	}
	return defs
}
