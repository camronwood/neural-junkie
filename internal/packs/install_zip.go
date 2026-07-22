package packs

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxPackZipBytes = 64 << 20 // 64 MiB (sidecar binaries can exceed 10 MiB)

var installHTTPClient = &http.Client{Timeout: 120 * time.Second}

// SetInstallHTTPClientForTests replaces the pack install HTTP client (tests only).
func SetInstallHTTPClientForTests(c *http.Client) (restore func()) {
	prev := installHTTPClient
	if c != nil {
		installHTTPClient = c
	}
	return func() { installHTTPClient = prev }
}

// InstallOfficialPack installs from catalog download_url (catalog-only; requires network).
func InstallOfficialPack(packID string) error {
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return fmt.Errorf("pack id required")
	}
	cat, err := FetchCatalog()
	if err != nil {
		return err
	}
	e := cat.CatalogEntryByID(packID)
	if e == nil {
		return fmt.Errorf("unknown pack %q", packID)
	}
	dlURL := strings.TrimSpace(e.DownloadURL)
	if dlURL == "" {
		return fmt.Errorf("pack %q has no download_url in catalog", packID)
	}
	wantVersion := strings.TrimSpace(e.Version)
	if err := installFromZipURL(packID, dlURL, wantVersion); err != nil {
		return fmt.Errorf("download pack %q: %w", packID, err)
	}
	return nil
}

func installFromZipURL(packID, rawURL, wantVersion string) error {
	if err := validateDownloadURL(rawURL); err != nil {
		return err
	}
	tmpZip, err := os.CreateTemp("", "nj-pack-*.zip")
	if err != nil {
		return err
	}
	zipPath := tmpZip.Name()
	defer os.Remove(zipPath)

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := installHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: %s (%s)", rawURL, resp.Status, strings.TrimSpace(string(body)))
	}
	n, err := io.Copy(tmpZip, io.LimitReader(resp.Body, maxPackZipBytes+1))
	if closeErr := tmpZip.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if n > maxPackZipBytes {
		return fmt.Errorf("pack zip exceeds %d bytes", maxPackZipBytes)
	}

	tmpDir, err := os.MkdirTemp("", "nj-pack-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZipSafe(zipPath, tmpDir); err != nil {
		return err
	}
	manifestDir, err := findManifestDir(tmpDir)
	if err != nil {
		return err
	}
	m, err := LoadManifest(manifestDir)
	if err != nil {
		return err
	}
	if m.ID != packID {
		return fmt.Errorf("pack manifest id %q does not match requested %q", m.ID, packID)
	}
	if wantVersion != "" && strings.TrimSpace(m.Version) != "" && m.Version != wantVersion {
		return fmt.Errorf("pack version %q does not match catalog %q", m.Version, wantVersion)
	}

	destRoot, err := UserPacksDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		return err
	}
	dest := filepath.Join(destRoot, packID)
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return copyDir(manifestDir, dest)
}

func validateDownloadURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid download url: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("pack download must use https, got %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	allowed := []string{
		"github.com",
		"raw.githubusercontent.com",
		"objects.githubusercontent.com",
	}
	ok := false
	for _, h := range allowed {
		if host == h || strings.HasSuffix(host, "."+h) {
			ok = true
			break
		}
	}
	if !ok && os.Getenv("NEURAL_JUNKIE_PACKS_ALLOW_TEST_HOSTS") == "1" {
		if host == "127.0.0.1" || host == "localhost" {
			ok = true
		}
	}
	if !ok {
		return fmt.Errorf("pack download host %q is not allowed", host)
	}
	return nil
}

func extractZipSafe(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		name := filepath.Join(destDir, f.Name)
		abs, err := filepath.Abs(name)
		if err != nil {
			return err
		}
		if abs != destAbs && !strings.HasPrefix(abs, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("zip slip: %q", f.Name)
		}
		name = abs
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(name, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode()&0777|0600)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, maxPackZipBytes))
		closeOut := out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOut != nil {
			return closeOut
		}
	}
	return nil
}

func findManifestDir(root string) (string, error) {
	if _, err := os.Stat(filepath.Join(root, "pack.yaml")); err == nil {
		return root, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if _, err := os.Stat(filepath.Join(dir, "pack.yaml")); err == nil {
			candidates = append(candidates, dir)
		}
	}
	switch len(candidates) {
	case 1:
		return candidates[0], nil
	case 0:
		return "", fmt.Errorf("pack.yaml not found in zip")
	default:
		return "", fmt.Errorf("multiple pack.yaml in zip; bundle one pack per archive")
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode&0777|0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}
