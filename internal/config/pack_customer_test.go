package config_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestInstallAndEnableCustomerPackRequiresLifeSciences(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	if err := cfg.InstallPack(config.PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	data, err := buildBrightestBioZip(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.InstallPackFromZip(data); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled("brightest-bio-lab", true); err == nil {
		t.Fatal("expected requires life-sciences enabled")
	}
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled("brightest-bio-lab", true); err != nil {
		t.Fatal(err)
	}
	if cfg.MCP.Biology.PythonExecutable != "python3" {
		t.Fatalf("overlay python: got %q", cfg.MCP.Biology.PythonExecutable)
	}
	if cfg.MCP.Biology.DefaultPanelProfile != "human-inflammatory-12plex-v1" {
		t.Fatalf("overlay panel profile: got %q", cfg.MCP.Biology.DefaultPanelProfile)
	}
	if !cfg.AnyPackCapability("secondary-analysis-customer") {
		t.Fatal("expected secondary-analysis-customer capability when brightest-bio-lab enabled")
	}
	ctxs, err := cfg.EnabledCustomerPackContexts()
	if err != nil {
		t.Fatal(err)
	}
	if len(ctxs) != 1 || ctxs[0].WorkspaceGuide == "" {
		t.Fatalf("expected workspace guide context: %+v", ctxs)
	}
	if err := cfg.SetPackEnabled("brightest-bio-lab", false); err != nil {
		t.Fatal(err)
	}
}

func buildBrightestBioZip(t *testing.T) ([]byte, error) {
	t.Helper()
	src := filepath.Join("..", "packs", "testdata", "brightest-bio-lab")
	if _, err := os.Stat(filepath.Join(src, "pack.yaml")); err != nil {
		t.Skip("brightest-bio-lab fixture missing")
	}
	zipPath := filepath.Join(t.TempDir(), "pack.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		return nil, err
	}
	w := zip.NewWriter(out)
	err = filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		fw, err := w.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
	if err != nil {
		out.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		out.Close()
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(zipPath)
}
