package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

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
	for _, a := range cfg.Agents {
		if a.Type == "biology" && a.Model != BioOllamaChatModel {
			t.Fatalf("biology agent model = %q, want %q", a.Model, BioOllamaChatModel)
		}
	}
}

func TestSyncAgentsFromPacksCAD(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackCAD)
	cfg.Packs.Enabled[PackCAD] = true
	cfg.SyncAgentsFromPacks()

	if !cfg.AgentTypeEnabled("cad") {
		t.Fatal("expected cad enabled")
	}
	foundChat := false
	foundTool := false
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == CadOllamaChatModel {
			foundChat = true
		}
		if m == CadOllamaToolModel {
			foundTool = true
		}
	}
	if !foundChat {
		t.Fatalf("expected %s in models_to_ensure", CadOllamaChatModel)
	}
	if !foundTool {
		t.Fatalf("expected %s in models_to_ensure", CadOllamaToolModel)
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

	if !cfg.AgentTypeEnabled("backend") {
		t.Fatal("expected backend enabled")
	}
	for _, typ := range devSpecialistTypes {
		if typ == "backend" {
			continue
		}
		if cfg.AgentTypeEnabled(typ) {
			t.Fatalf("expected %s disabled in slim default room", typ)
		}
	}
	foundCoder := false
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == UtilityOllamaModel {
			foundCoder = true
		}
		if m == DevOllamaCodeModel {
			t.Fatalf("did not expect %s in models_to_ensure", DevOllamaCodeModel)
		}
	}
	if !foundCoder {
		t.Fatalf("expected %s in models_to_ensure", UtilityOllamaModel)
	}
}

func TestSyncAgentsFromPacksPreservesDisabledSpecialist(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.Agents = []AgentConfig{
		{Type: "backend", Name: "BackendEngineer", Enabled: true},
		{Type: "frontend", Name: "FrontendEngineer", Enabled: false},
	}
	cfg.SyncAgentsFromPacks()
	if !cfg.AgentTypeEnabled("backend") {
		t.Fatal("expected backend still enabled")
	}
	if cfg.AgentTypeEnabled("frontend") {
		t.Fatal("sync must not force-enable a disabled specialist")
	}
}

func TestApplySlimDefaultRoom(t *testing.T) {
	t.Setenv("NJ_DEFAULT_ROOM", "")
	cfg := DefaultConfig()
	cfg.Agents = []AgentConfig{
		{Type: "assistant", Name: "Assistant", Enabled: true, Model: DevOllamaCodeModel},
		{Type: "backend", Name: "BackendEngineer", Enabled: true, Model: DevOllamaCodeModel},
		{Type: "frontend", Name: "FrontendEngineer", Enabled: true},
		{Type: "architecture", Name: "SoftwareArchitect", Enabled: true},
	}
	cfg.AI.Providers[0].Model = DevOllamaCodeModel
	cfg.Ollama.ModelsToEnsure = []string{DevOllamaCodeModel, UtilityOllamaModel}
	if !cfg.applySlimDefaultRoom() {
		t.Fatal("expected persist when default_room was empty")
	}
	if cfg.Packs.DefaultRoom != DefaultRoomSlim {
		t.Fatalf("default_room = %q", cfg.Packs.DefaultRoom)
	}
	if !cfg.AgentTypeEnabled("backend") || !cfg.AgentTypeEnabled("assistant") {
		t.Fatal("expected Assistant + BackendEngineer enabled")
	}
	if cfg.AgentTypeEnabled("frontend") || cfg.AgentTypeEnabled("architecture") {
		t.Fatal("expected extra specialists disabled")
	}
	for _, a := range cfg.Agents {
		if a.Type == "backend" || a.Type == "assistant" {
			if a.Model != UtilityOllamaModel {
				t.Fatalf("%s model = %q, want %q", a.Type, a.Model, UtilityOllamaModel)
			}
		}
	}
	if cfg.AI.Providers[0].Model != UtilityOllamaModel {
		t.Fatalf("provider model = %q", cfg.AI.Providers[0].Model)
	}
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == DevOllamaCodeModel {
			t.Fatal("27b should not remain in models_to_ensure")
		}
	}
}

func TestApplySlimDefaultRoomSkipsWhenAlreadySet(t *testing.T) {
	t.Setenv("NJ_DEFAULT_ROOM", "")
	cfg := DefaultConfig()
	cfg.Packs.DefaultRoom = DefaultRoomSlim
	cfg.Agents = []AgentConfig{
		{Type: "backend", Name: "BackendEngineer", Enabled: true},
		{Type: "frontend", Name: "FrontendEngineer", Enabled: true},
	}
	if cfg.applySlimDefaultRoom() {
		t.Fatal("must not re-apply slim after default_room is set")
	}
	if !cfg.AgentTypeEnabled("frontend") {
		t.Fatal("user-enabled specialist must stick after the one-shot slim migrate")
	}
}

func TestSyncAgentsFromPacksPreservesCustomTypeOccupant(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.Agents = []AgentConfig{
		{Type: "backend", Name: "SwitchTarget", Enabled: true, ProviderID: "ollama-local"},
	}
	cfg.SyncAgentsFromPacks()

	hasBackend := false
	hasSwitch := false
	for _, a := range cfg.Agents {
		if a.Name == "BackendEngineer" && a.Enabled {
			hasBackend = true
		}
		if a.Name == "SwitchTarget" && a.Enabled {
			hasSwitch = true
		}
	}
	if !hasBackend {
		t.Fatal("expected BackendEngineer from pack when custom backend agent occupies type slot")
	}
	if !hasSwitch {
		t.Fatal("expected SwitchTarget preserved")
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
	hasDatabase := false
	hasRust := false
	for _, p := range presets {
		if p.Slug == "architecture" && p.FromPack == PackSoftwareDevelopment {
			hasArchitecture = true
		}
		if p.Slug == "code-review" {
			t.Fatal("code-review preset should be removed from software-development pack")
		}
		if p.Slug == "database" && p.FromPack == PackSoftwareDevelopment {
			hasDatabase = true
		}
		if p.Slug == "rust" && p.FromPack == PackSoftwareDevelopment {
			hasRust = true
		}
	}
	if !hasArchitecture {
		t.Fatal("expected architecture preset from software-development pack")
	}
	if !hasDatabase {
		t.Fatal("expected database preset from software-development pack")
	}
	if !hasRust {
		t.Fatal("expected rust preset from software-development pack")
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
	if cfg.PresetExpertAllowed("cad") {
		t.Fatal("cad should be blocked when CAD pack off")
	}
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	if !cfg.PresetExpertAllowed("rust") {
		t.Fatal("legacy rust should be allowed when dev pack on")
	}
	if cfg.PresetExpertAllowed("code-review") {
		t.Fatal("code-review should remain blocked (review is a core agent behavior)")
	}
	if cfg.PresetExpertAllowed("maps") {
		t.Fatal("maps should be blocked (ability pack on Assistant)")
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

func TestSetLayoutOwner(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	installTestPack(t, cfg, PackIDE)
	if err := cfg.SetPackEnabled(PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(PackIDE, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetLayoutOwner(PackIDE); err != nil {
		t.Fatal(err)
	}
	if cfg.LayoutOwnerPackID() != PackIDE {
		t.Fatalf("expected layout owner ide, got %q", cfg.LayoutOwnerPackID())
	}
	if cfg.LayoutProfile() != "ide" {
		t.Fatalf("expected ide layout profile, got %q", cfg.LayoutProfile())
	}
}

func TestSetLayoutOwnerRejectsDisabled(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	installTestPack(t, cfg, PackSoftwareDevelopment)
	_ = cfg.SetPackEnabled(PackLifeSciences, true)
	if err := cfg.SetLayoutOwner(PackSoftwareDevelopment); err == nil {
		t.Fatal("expected error for disabled pack")
	}
}

func TestSetLayoutOwnerRejectsUnknown(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.SetLayoutOwner("nonexistent-pack"); err == nil {
		t.Fatal("expected error for unknown pack")
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
	installTestPack(t, cfg, PackIDE)
	_ = cfg.SetPackEnabled(PackIDE, true)
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
	setupTestOfficialPackCatalog(t)
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

func TestMigrateIdePackIfNeeded(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.Packs.LayoutOwner = PackSoftwareDevelopment
	cfg.migrateIdePackIfNeeded()
	if !cfg.IsPackEnabled(PackIDE) {
		t.Fatal("expected ide pack enabled when software-development was on")
	}
	if !cfg.IsPackInstalled(PackIDE) {
		t.Fatal("expected ide pack installed")
	}
	if cfg.LayoutOwnerPackID() != PackIDE {
		t.Fatalf("expected layout owner ide, got %q", cfg.LayoutOwnerPackID())
	}
}

func TestMigrateInstalledPacksBothOn(t *testing.T) {
	setupTestOfficialPackCatalog(t)
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
	if cfg.LayoutOwnerPackID() != PackIDE {
		t.Fatalf("expected ide layout owner when both on, got %q", cfg.LayoutOwnerPackID())
	}
	if !cfg.IsPackEnabled(PackIDE) {
		t.Fatal("expected ide pack enabled after migration when software-development was on")
	}
}

func TestListPackCatalogStatusLoRAAdapterCount(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSpecialistTuning)
	rows, err := cfg.ListPackCatalogStatus()
	if err != nil {
		t.Fatal(err)
	}
	var tuningRow *PackCatalogStatus
	for i := range rows {
		if rows[i].ID == PackSpecialistTuning {
			tuningRow = &rows[i]
			break
		}
	}
	if tuningRow == nil {
		t.Fatal("specialist-tuning not in catalog")
	}
	if tuningRow.LoRAAdapterCount != 9 {
		t.Fatalf("expected 9 lora adapters, got %d", tuningRow.LoRAAdapterCount)
	}
	bases := map[string]struct{}{}
	for _, b := range tuningRow.LoRABaseTags {
		bases[b] = struct{}{}
	}
	if _, ok := bases["llama3:8b"]; !ok {
		t.Fatalf("expected llama3:8b in lora_base_tags: %v", tuningRow.LoRABaseTags)
	}
	if _, ok := bases["llama3.2:3b"]; !ok {
		t.Fatalf("expected llama3.2:3b in lora_base_tags: %v", tuningRow.LoRABaseTags)
	}
	if _, ok := bases["mistral:7b"]; !ok {
		t.Fatalf("expected mistral:7b in lora_base_tags: %v", tuningRow.LoRABaseTags)
	}
	if _, ok := bases["qwen2.5-coder:14b"]; ok {
		t.Fatalf("qwen2.5-coder:14b should not be a LoRA base tag: %v", tuningRow.LoRABaseTags)
	}
}

func TestMigrateSpecialistTuningForLoRAUsers(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	cfg.Ollama.ModelsToEnsure = []string{"nj-security:14b", "qwen2.5-coder:14b"}
	cfg.Agents = append(cfg.Agents, AgentConfig{
		Type:       "security",
		Name:       "SecurityReviewer",
		Enabled:    true,
		ProviderID: "ollama-local",
		Model:      "nj-security:14b",
	})
	installTestPack(t, cfg, PackSoftwareDevelopment)
	_ = cfg.SetPackEnabled(PackSoftwareDevelopment, true)

	cfg.MigrateSpecialistTuningForLoRAUsers()

	if !cfg.IsPackInstalled(PackSpecialistTuning) {
		t.Fatal("expected specialist-tuning installed")
	}
	if !cfg.IsPackEnabled(PackSpecialistTuning) {
		t.Fatal("expected specialist-tuning enabled after nj-* migration")
	}
}
