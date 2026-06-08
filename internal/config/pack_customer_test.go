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
	data, err := buildCustomerLabPackZip(t)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.InstallPackFromZip(data); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled("customer-lab-pack", true); err == nil {
		t.Fatal("expected requires life-sciences enabled")
	}
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if cfg.AnyPackCapability("scan-summary-api") || cfg.AnyPackCapability("scan-analysis-viewer") {
		t.Fatal("life-sciences alone should not grant scan viewers")
	}
	if err := cfg.SetPackEnabled("customer-lab-pack", true); err != nil {
		t.Fatal(err)
	}
	if cfg.MCP.Biology.PythonExecutable != "python3" {
		t.Fatalf("overlay python: got %q", cfg.MCP.Biology.PythonExecutable)
	}
	if cfg.MCP.Biology.DefaultPanelProfile != "human-inflammatory-12plex-v1" {
		t.Fatalf("overlay panel profile: got %q", cfg.MCP.Biology.DefaultPanelProfile)
	}
	if !cfg.AnyPackCapability("secondary-analysis-api") {
		t.Fatal("expected secondary-analysis-api capability when customer-lab-pack enabled")
	}
	if !cfg.AnyPackCapability("secondary-analysis-python") {
		t.Fatal("expected secondary-analysis-python capability when customer-lab-pack enabled")
	}
	if !cfg.AnyPackCapability("scan-summary-api") {
		t.Fatal("expected scan-summary-api capability when customer-lab-pack enabled")
	}
	if !cfg.AnyPackCapability("scan-analysis-viewer") {
		t.Fatal("expected scan-analysis-viewer capability when customer-lab-pack enabled")
	}
	ctxs, err := cfg.EnabledCustomerPackContexts()
	if err != nil {
		t.Fatal(err)
	}
	if len(ctxs) != 1 || ctxs[0].WorkspaceGuide == "" {
		t.Fatalf("expected workspace guide context: %+v", ctxs)
	}
	if err := cfg.SetPackEnabled("customer-lab-pack", false); err != nil {
		t.Fatal(err)
	}
}

func buildCustomerLabPackZip(t *testing.T) ([]byte, error) {
	t.Helper()
	src := filepath.Join("..", "packs", "testdata", "customer-lab-pack")
	if _, err := os.Stat(filepath.Join(src, "pack.yaml")); err != nil {
		t.Skip("customer-lab-pack fixture missing")
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
