package cad

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// VersionMeta describes one saved CAD snapshot.
type VersionMeta struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	CreatedAt time.Time         `json:"created_at"`
	Params    map[string]string `json:"params,omitempty"`
}

type versionRecord struct {
	VersionMeta
	SCADRel string `json:"scad_rel"`
	STLRel  string `json:"stl_rel,omitempty"`
}

// SaveVersion snapshots SCAD (and optional STL) under project versions/.
func SaveVersion(paths ProjectPaths, label string, params map[string]string, scadContent string, stlPath string) (VersionMeta, error) {
	if err := os.MkdirAll(paths.VersionsDir, 0755); err != nil {
		return VersionMeta{}, err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	dir := filepath.Join(paths.VersionsDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return VersionMeta{}, err
	}
	scadRel := filepath.Join("versions", id, "model.scad")
	scadAbs := filepath.Join(paths.Root, scadRel)
	if err := WriteSCADFile(scadAbs, scadContent); err != nil {
		return VersionMeta{}, err
	}
	rec := versionRecord{
		VersionMeta: VersionMeta{
			ID:        id,
			Label:     strings.TrimSpace(label),
			CreatedAt: time.Now().UTC(),
			Params:    params,
		},
		SCADRel: scadRel,
	}
	if label == "" {
		rec.Label = id
	}
	if stlPath != "" {
		if _, err := os.Stat(stlPath); err == nil {
			stlRel := filepath.Join("versions", id, "preview.stl")
			if err := CopyFile(stlPath, filepath.Join(paths.Root, stlRel)); err == nil {
				rec.STLRel = stlRel
			}
		}
	}
	metaPath := filepath.Join(dir, "meta.json")
	b, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(metaPath, b, 0644); err != nil {
		return VersionMeta{}, err
	}
	return rec.VersionMeta, nil
}

// ListVersions returns snapshots newest-first.
func ListVersions(paths ProjectPaths) ([]VersionMeta, error) {
	entries, err := os.ReadDir(paths.VersionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []VersionMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metaPath := filepath.Join(paths.VersionsDir, e.Name(), "meta.json")
		b, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var rec versionRecord
		if json.Unmarshal(b, &rec) != nil {
			continue
		}
		out = append(out, rec.VersionMeta)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

// RestoreVersion copies version SCAD back to project model.scad.
func RestoreVersion(paths ProjectPaths, versionID string) (string, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return "", fmt.Errorf("version id required")
	}
	metaPath := filepath.Join(paths.VersionsDir, versionID, "meta.json")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		return "", fmt.Errorf("version not found: %w", err)
	}
	var rec versionRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return "", err
	}
	src := filepath.Join(paths.Root, rec.SCADRel)
	content, err := ReadSCADFile(src)
	if err != nil {
		return "", err
	}
	if err := WriteSCADFile(paths.SCADPath, content); err != nil {
		return "", err
	}
	if rec.STLRel != "" {
		stlSrc := filepath.Join(paths.Root, rec.STLRel)
		if _, err := os.Stat(stlSrc); err == nil {
			_ = CopyFile(stlSrc, paths.STLPath)
		}
	}
	return content, nil
}
