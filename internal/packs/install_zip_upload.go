package packs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

	destRoot, err := UserPacksDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return nil, err
	}
	dest := filepath.Join(destRoot, m.ID)
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := copyDir(manifestDir, dest); err != nil {
		return nil, err
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
