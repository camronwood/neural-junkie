package packs

import (
	"bytes"
	"fmt"
	"os"
)

// InstallFromZipBytes validates and installs a pack zip to ~/.neural-junkie/packs/<id>/.
func InstallFromZipBytes(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty pack zip")
	}
	if len(data) > maxPackZipBytes {
		return nil, fmt.Errorf("pack zip exceeds %d bytes", maxPackZipBytes)
	}

	tmpZip, err := os.CreateTemp("", "nj-pack-upload-*.zip")
	if err != nil {
		return nil, err
	}
	zipPath := tmpZip.Name()
	defer os.Remove(zipPath)
	if _, err := tmpZip.Write(data); err != nil {
		tmpZip.Close()
		return nil, err
	}
	if err := tmpZip.Close(); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "nj-pack-extract-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZipSafe(zipPath, tmpDir); err != nil {
		return nil, err
	}
	manifestDir, err := findManifestDir(tmpDir)
	if err != nil {
		return nil, err
	}
	m, err := LoadManifest(manifestDir)
	if err != nil {
		return nil, err
	}
	if IsOfficialPackID(m.ID) {
		return nil, fmt.Errorf("pack id %q collides with an official catalog pack", m.ID)
	}

	destRoot, err := UserPacksDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return nil, err
	}
	dest, err := PackInstallDest(destRoot, m.ID)
	if err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(destRoot, ".nj-pack-stage-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if staging != "" {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := copyDir(manifestDir, staging); err != nil {
		return nil, err
	}
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Rename(staging, dest); err != nil {
		// Fallback when rename crosses devices: copy then remove staging (defer).
		if copyErr := copyDir(manifestDir, dest); copyErr != nil {
			return nil, fmt.Errorf("install pack: rename: %v; copy: %w", err, copyErr)
		}
	} else {
		staging = "" // renamed into place; skip deferred cleanup
	}
	return LoadManifest(dest)
}

// InstallFromZipReader installs from an in-memory zip buffer.
func InstallFromZipReader(r *bytes.Reader) (*Manifest, error) {
	data := make([]byte, r.Len())
	if _, err := r.Read(data); err != nil {
		return nil, err
	}
	return InstallFromZipBytes(data)
}
