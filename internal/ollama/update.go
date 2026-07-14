package ollama

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NeuralJunkieOllamaRuntimeDir is the user-writable bundled runtime (prefer over app resources).
func NeuralJunkieOllamaRuntimeDir() string {
	triple := HostTriple()
	if triple == "" {
		triple = "unknown"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".neural-junkie", "ollama-runtime", triple)
	}
	return filepath.Join(os.TempDir(), "neural-junkie-ollama-runtime", triple)
}

// UserRuntimeBinary returns ollama in the user-updated runtime dir, if present.
func UserRuntimeBinary() string {
	return bundledBinaryInDir(NeuralJunkieOllamaRuntimeDir())
}

// UpdateSupported reports whether NJ can attempt an in-app update.
func UpdateSupported(status InstallStatus) bool {
	if status.Bundled || UserRuntimeBinary() != "" || BundledBinaryPath() != "" || DevBundledBinaryPath() != "" {
		return HostTriple() != ""
	}
	return AutoInstallSupported()
}

// UpdateOllama upgrades bundled user-runtime or re-runs the system installer.
func (m *Manager) UpdateOllama(ctx context.Context, onProgress func(string)) error {
	status := m.DetectInstallation()
	managed := status.Bundled || UserRuntimeBinary() != "" || BundledBinaryPath() != "" || DevBundledBinaryPath() != ""
	if managed {
		return m.updateBundledRuntime(ctx, onProgress)
	}
	if !status.Installed {
		return m.InstallOllama(ctx, onProgress)
	}
	if onProgress != nil {
		onProgress(fmt.Sprintf("Updating system Ollama toward %s...", RecommendedOllamaVersion))
	}
	if err := runPlatformOllamaInstall(ctx, onProgress); err != nil {
		return err
	}
	after := m.DetectInstallation()
	ver, _ := ParseOllamaVersion(after.Version)
	if ver != "" && NeedsUpdate(ver) {
		return fmt.Errorf("Ollama is still %s after update (want %s+); install manually from https://ollama.com/download", ver, RecommendedOllamaVersion)
	}
	if onProgress != nil {
		onProgress("Ollama update complete")
	}
	return nil
}

func (m *Manager) updateBundledRuntime(ctx context.Context, onProgress func(string)) error {
	triple := HostTriple()
	if triple == "" {
		return fmt.Errorf("unsupported platform for bundled Ollama update")
	}
	dest := NeuralJunkieOllamaRuntimeDir()
	if onProgress != nil {
		onProgress(fmt.Sprintf("Downloading Ollama %s for %s...", RecommendedOllamaTag, triple))
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	// Stop managed serve before replacing binaries.
	_ = m.StopServer()

	tmpRoot, err := os.MkdirTemp("", "nj-ollama-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRoot)

	if err := downloadAndExtractOllama(ctx, triple, tmpRoot, onProgress); err != nil {
		return err
	}

	binSrc := bundledBinaryInDir(tmpRoot)
	if binSrc == "" {
		// Archive may nest one level.
		_ = filepath.Walk(tmpRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return err
			}
			base := strings.ToLower(info.Name())
			if base == "ollama" || base == "ollama.exe" {
				binSrc = path
				return io.EOF
			}
			return nil
		})
	}
	if binSrc == "" {
		return fmt.Errorf("downloaded archive did not contain an ollama binary")
	}

	staging := dest + ".staging"
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}

	// Copy extracted tree into staging (preserve llama-server etc. when present).
	if err := copyDir(tmpRoot, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	binInStaging := bundledBinaryInDir(staging)
	if binInStaging == "" {
		_ = os.RemoveAll(staging)
		return fmt.Errorf("failed to stage ollama binary")
	}
	_ = os.Chmod(binInStaging, 0o755)

	backup := dest + ".bak"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, backup); err != nil {
			_ = os.RemoveAll(staging)
			return fmt.Errorf("backup old runtime: %w", err)
		}
	}
	if err := os.Rename(staging, dest); err != nil {
		_ = os.Rename(backup, dest)
		return fmt.Errorf("activate new runtime: %w", err)
	}
	_ = os.RemoveAll(backup)

	if onProgress != nil {
		onProgress("Starting updated Ollama...")
	}
	startCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := m.StartServer(startCtx); err != nil && onProgress != nil {
		onProgress(fmt.Sprintf("Updated binary installed; start failed: %v", err))
	}
	if onProgress != nil {
		onProgress(fmt.Sprintf("Ollama updated to %s", RecommendedOllamaVersion))
	}
	return nil
}

func downloadAndExtractOllama(ctx context.Context, triple, dest string, onProgress func(string)) error {
	base := "https://github.com/ollama/ollama/releases/download/" + RecommendedOllamaTag
	client := &http.Client{Timeout: 30 * time.Minute}

	switch triple {
	case "aarch64-apple-darwin", "x86_64-apple-darwin":
		archive := filepath.Join(dest, "ollama.tgz")
		if err := downloadFile(ctx, client, base+"/ollama-darwin.tgz", archive, onProgress); err != nil {
			return err
		}
		return extractTarGz(archive, dest)
	case "x86_64-unknown-linux-gnu":
		archive := filepath.Join(dest, "ollama.tar.zst")
		if err := downloadFile(ctx, client, base+"/ollama-linux-amd64.tar.zst", archive, onProgress); err != nil {
			return err
		}
		return extractTarZst(archive, dest)
	case "x86_64-pc-windows-msvc":
		archive := filepath.Join(dest, "ollama.zip")
		if err := downloadFile(ctx, client, base+"/ollama-windows-amd64.zip", archive, onProgress); err != nil {
			return err
		}
		return extractZip(archive, dest)
	default:
		return fmt.Errorf("unsupported triple %s", triple)
	}
}

func downloadFile(ctx context.Context, client *http.Client, url, dest string, onProgress func(string)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "neural-junkie-ollama-updater")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	var written int64
	buf := make([]byte, 1024*1024)
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if onProgress != nil && time.Since(last) > 2*time.Second {
				onProgress(fmt.Sprintf("Downloaded %.1f MB...", float64(written)/1e6))
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return nil
}

func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return untar(gz, dest)
}

func extractTarZst(archive, dest string) error {
	// Prefer system zstd; fall back to python zstandard.
	outTar := filepath.Join(dest, "ollama.tar")
	if _, err := exec.LookPath("zstd"); err == nil {
		cmd := exec.Command("zstd", "-d", archive, "-o", outTar)
		cmd.Dir = dest
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("zstd decompress: %w (%s)", err, string(out))
		}
	} else {
		py := `import pathlib,zstandard,sys
raw=pathlib.Path(sys.argv[1]).read_bytes()
pathlib.Path(sys.argv[2]).write_bytes(zstandard.ZstdDecompressor().decompress(raw))`
		cmd := exec.Command("python3", "-c", py, archive, outTar)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("need zstd or python3+zstandard to extract linux Ollama: %w (%s)", err, string(out))
		}
	}
	f, err := os.Open(outTar)
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(outTar)
	return untar(f, dest)
}

func extractZip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, zf := range r.File {
		target := filepath.Join(dest, zf.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(dest) {
			return fmt.Errorf("illegal zip path %q", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, zf.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(dest) {
			return fmt.Errorf("illegal tar path %q", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			out.Close()
			if copyErr != nil {
				return copyErr
			}
		}
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
		// Skip archive leftovers
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".tgz") || strings.HasSuffix(base, ".tar") ||
			strings.HasSuffix(base, ".zst") || strings.HasSuffix(base, ".zip") {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		out.Close()
		return copyErr
	})
}
