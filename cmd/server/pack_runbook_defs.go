package main

import (
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func serverPackRunbookDefinitions() []runbooklibrary.RunbookDefinition {
	if appConfig == nil {
		return nil
	}
	statuses := appConfig.ListPackStatus().Packs
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
		m, err := appConfig.InstalledPackManifestByID(st.ID)
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
	return defs
}
