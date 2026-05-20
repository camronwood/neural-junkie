package config

import "testing"

func TestWizardProfileLifeSciences(t *testing.T) {
	p := WizardProfileFor(WizardTrackLifeSciences)
	if p.OllamaModel != BioOllamaChatModel {
		t.Fatalf("model = %q", p.OllamaModel)
	}
	if len(p.DefaultAgents) != 2 {
		t.Fatalf("agents = %d", len(p.DefaultAgents))
	}
	if p.DefaultAgents[0].Type != "biology" {
		t.Fatalf("first agent type = %s", p.DefaultAgents[0].Type)
	}
}

func TestWizardProfileDeveloper(t *testing.T) {
	p := WizardProfileFor(WizardTrackDeveloper)
	if p.OllamaModel != DevOllamaCodeModel {
		t.Fatalf("model = %q", p.OllamaModel)
	}
}

func TestWizardProfileGeneral(t *testing.T) {
	p := WizardProfileFor(WizardTrackGeneral)
	if p.OllamaModel != UtilityOllamaModel {
		t.Fatalf("model = %q", p.OllamaModel)
	}
	if len(p.DefaultAgents) != 1 || p.DefaultAgents[0].Type != "assistant" {
		t.Fatalf("agents = %+v", p.DefaultAgents)
	}
}

func TestApplyWizardProfileDeveloperPack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ApplyWizardProfile(WizardTrackDeveloper, true)
	if !cfg.IsPackEnabled(PackSoftwareDevelopment) {
		t.Fatal("expected software-development pack on")
	}
	if !cfg.AgentTypeEnabled("backend") {
		t.Fatal("expected backend from pack sync")
	}
}

func TestApplyWizardProfileGeneralPack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ApplyWizardProfile(WizardTrackGeneral, true)
	if cfg.IsPackEnabled(PackSoftwareDevelopment) {
		t.Fatal("expected software-development pack off")
	}
	if cfg.AgentTypeEnabled("backend") {
		t.Fatal("expected no backend specialist")
	}
}
