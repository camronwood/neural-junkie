package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func installTestPack(t *testing.T, cfg *Config, packID string) {
	t.Helper()
	if err := cfg.InstallPack(packID); err != nil {
		t.Fatal(err)
	}
}

func TestSyncAgentsFromPacksLifeSciences(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	cfg.Packs.Enabled[PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()

	if !cfg.AgentTypeEnabled("biology") {
		t.Fatal("expected biology enabled")
	}
	foundChat := false
	foundTool := false
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == BioOllamaChatModel {
			foundChat = true
		}
		if m == BioOllamaToolModel {
			foundTool = true
		}
	}
	if !foundChat {
		t.Fatalf("expected %s in models_to_ensure", BioOllamaChatModel)
	}
	if !foundTool {
		t.Fatalf("expected %s in models_to_ensure", BioOllamaToolModel)
	}
}

func TestSyncAgentsFromPacksDisabled(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	cfg.Agents = append(cfg.Agents, AgentConfig{Type: "biology", Name: "BiologyExpert", Enabled: true})
	cfg.Packs.Enabled[PackLifeSciences] = false
	cfg.SyncAgentsFromPacks()

	if cfg.AgentTypeEnabled("biology") {
		t.Fatal("expected biology disabled when pack off")
	}
}

func TestAvailableExpertPresets(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	cfg.Packs.Enabled[PackLifeSciences] = true
	presets := cfg.AvailableExpertPresets()
	hasBio := false
	for _, p := range presets {
		if p.Slug == "biology" && p.FromPack == PackLifeSciences {
			hasBio = true
		}
	}
	if !hasBio {
		t.Fatal("expected biology preset from life-sciences pack")
	}
}

func TestSyncAgentsFromPacksSoftwareDevelopment(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.Packs.LayoutOwner = PackSoftwareDevelopment
	cfg.SyncAgentsFromPacks()

	for _, typ := range devSpecialistTypes {
		if !cfg.AgentTypeEnabled(typ) {
			t.Fatalf("expected %s enabled", typ)
		}
	}
	foundCoder := false
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == DevOllamaCodeModel {
			foundCoder = true
		}
	}
	if !foundCoder {
		t.Fatalf("expected %s in models_to_ensure", DevOllamaCodeModel)
	}
}

func TestSyncAgentsFromPacksSoftwareDevelopmentDisabled(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Agents = append(cfg.Agents, AgentConfig{Type: "backend", Name: "BackendEngineer", Enabled: true})
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	cfg.SyncAgentsFromPacks()

	if cfg.AgentTypeEnabled("backend") {
		t.Fatal("expected backend disabled when software-development pack off")
	}
}

func TestAvailableExpertPresetsDevPack(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	presets := cfg.AvailableExpertPresets()
	hasArchitecture := false
	hasCodeReview := false
	hasDatabase := false
	for _, p := range presets {
		if p.Slug == "architecture" && p.FromPack == PackSoftwareDevelopment {
			hasArchitecture = true
		}
		if p.Slug == "code-review" && p.FromPack == PackSoftwareDevelopment {
			hasCodeReview = true
		}
		if p.Slug == "database" && p.FromPack == PackSoftwareDevelopment {
			hasDatabase = true
		}
		if p.Slug == "rust" {
			t.Fatalf("legacy preset %q should not appear in software-development pack defaults", p.Slug)
		}
	}
	if !hasArchitecture {
		t.Fatal("expected architecture preset from software-development pack")
	}
	if !hasCodeReview {
		t.Fatal("expected code-review preset from software-development pack")
	}
	if !hasDatabase {
		t.Fatal("expected database preset from software-development pack")
	}
}

func TestPresetExpertAllowed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	if !cfg.PresetExpertAllowed("assistant") {
		t.Fatal("assistant should always be allowed")
	}
	if cfg.PresetExpertAllowed("rust") {
		t.Fatal("rust should be blocked when dev pack off")
	}
	if cfg.PresetExpertAllowed("code-review") {
		t.Fatal("code-review should be blocked when dev pack off")
	}
	if cfg.PresetExpertAllowed("biology") {
		t.Fatal("biology should be blocked when life-sciences off")
	}
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	if !cfg.PresetExpertAllowed("rust") {
		t.Fatal("legacy rust should be allowed when dev pack on")
	}
	if !cfg.PresetExpertAllowed("code-review") {
		t.Fatal("code-review should be allowed when dev pack on")
	}
	if !cfg.PresetExpertAllowed("guitar") {
		t.Fatal("custom slugs should be allowed")
	}
}

func TestMigrateSoftwareDevelopmentPackIfNeeded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Agents = []AgentConfig{{Type: "backend", Name: "BackendEngineer", Enabled: true}}
	delete(cfg.Packs.Enabled, PackSoftwareDevelopment)
	cfg.migrateSoftwareDevelopmentPackIfNeeded()
	if !cfg.Packs.Enabled[PackSoftwareDevelopment] {
		t.Fatal("expected migration to enable software-development pack")
	}
	if !cfg.IsPackInstalled(PackSoftwareDevelopment) {
		t.Fatal("expected migration to install software-development pack")
	}
}

func TestSpecialistShouldBeRunningPackOffOverridesConfig(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Agents = []AgentConfig{{Type: "backend", Name: "BackendEngineer", Enabled: true}}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	if cfg.SpecialistShouldBeRunning("backend") {
		t.Fatal("expected backend not running when software-development pack off even if config enabled")
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	if !cfg.SpecialistShouldBeRunning("backend") {
		t.Fatal("expected backend running when pack on and config enabled")
	}
}

func TestSetPackEnabledMultiPack(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	installTestPack(t, cfg, PackSoftwareDevelopment)
	if err := cfg.SetPackEnabled(PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(PackSoftwareDevelopment, true); err != nil {
		t.Fatal(err)
	}
	if !cfg.IsPackEnabled(PackLifeSciences) || !cfg.IsPackEnabled(PackSoftwareDevelopment) {
		t.Fatal("expected both packs enabled")
	}
	if cfg.LayoutOwnerPackID() != PackLifeSciences {
		t.Fatalf("expected layout owner life-sciences (first enabled), got %q", cfg.LayoutOwnerPackID())
	}
}

func TestLayoutOwnerClearedWhenAllDisabled(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	_ = cfg.SetPackEnabled(PackSoftwareDevelopment, true)
	_ = cfg.SetPackEnabled(PackSoftwareDevelopment, false)
	if cfg.LayoutOwnerPackID() != "" {
		t.Fatalf("expected empty layout owner, got %q", cfg.LayoutOwnerPackID())
	}
}

func TestAnyPackCapability(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	_ = cfg.SetPackEnabled(PackSoftwareDevelopment, true)
	if !cfg.AnyPackCapability("git-rest") {
		t.Fatal("expected git-rest capability")
	}
}

func TestDefaultConfigNoPacksInstalled(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Packs.Installed) != 0 {
		t.Fatalf("expected no installed packs, got %v", cfg.Packs.Installed)
	}
	if cfg.IsPackEnabled(PackSoftwareDevelopment) {
		t.Fatal("expected dev pack off in defaults")
	}
}

func TestInstallAndUninstallPack(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.InstallPack(PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	if !cfg.IsPackInstalled(PackLifeSciences) {
		t.Fatal("expected installed")
	}
	dir, _ := packs.InstalledPackDir(PackLifeSciences)
	if _, err := os.Stat(filepath.Join(dir, "pack.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := cfg.UninstallPack(PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	if cfg.IsPackInstalled(PackLifeSciences) {
		t.Fatal("expected uninstalled")
	}
}

func TestSetPackEnabledRequiresInstall(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.SetPackEnabled(PackLifeSciences, true); err == nil {
		t.Fatal("expected error enabling without install")
	}
}

func TestMigrateInstalledPacksBothOn(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.Packs.Enabled[PackLifeSciences] = true
	cfg.MigrateInstalledPacks()
	if !cfg.IsPackInstalled(PackSoftwareDevelopment) || !cfg.IsPackInstalled(PackLifeSciences) {
		t.Fatal("expected both packs installed after migration")
	}
	if !cfg.IsPackEnabled(PackSoftwareDevelopment) || !cfg.IsPackEnabled(PackLifeSciences) {
		t.Fatal("expected both packs still enabled")
	}
	if cfg.LayoutOwnerPackID() != PackSoftwareDevelopment {
		t.Fatalf("expected dev layout owner when both on, got %q", cfg.LayoutOwnerPackID())
	}
}

func TestListPackCatalogStatusLoRAAdapterCount(t *testing.T) {
	cfg := DefaultConfig()
	rows, err := cfg.ListPackCatalogStatus()
	if err != nil {
		t.Fatal(err)
	}
	var devRow *PackCatalogStatus
	for i := range rows {
		if rows[i].ID == PackSoftwareDevelopment {
			devRow = &rows[i]
			break
		}
	}
	if devRow == nil {
		t.Fatal("software-development not in catalog")
	}
	if devRow.LoRAAdapterCount != 3 {
		t.Fatalf("expected 3 lora adapters, got %d", devRow.LoRAAdapterCount)
	}
	if len(devRow.LoRABaseTags) != 1 || devRow.LoRABaseTags[0] != "qwen2.5-coder:14b" {
		t.Fatalf("unexpected lora_base_tags: %v", devRow.LoRABaseTags)
	}
}
