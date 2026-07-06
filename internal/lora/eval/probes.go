package eval

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProbeFile is a pack-owned eval probe definition (assets/eval/*.yaml).
type ProbeFile struct {
	AgentType string     `yaml:"agent_type,omitempty"`
	Questions []Question `yaml:"questions"`
}

// LoadProbeFile reads questions from a YAML probe file.
func LoadProbeFile(path string) ([]Question, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pf ProbeFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return pf.Questions, nil
}

// LoadProbeFileFromPack resolves a pack-relative eval_probes path against packDir.
func LoadProbeFileFromPack(packDir, relPath string) ([]Question, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" || packDir == "" {
		return nil, nil
	}
	path := relPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(packDir, relPath)
	}
	return LoadProbeFile(path)
}

// QuestionsForAgentType picks probes: pack file when present, else defaults.
func QuestionsForAgentType(agentType, agentID, packDir, probeRel string) []Question {
	if probes, err := LoadProbeFileFromPack(packDir, probeRel); err == nil && len(probes) > 0 {
		return probes
	}
	if agentType == "repo" || strings.Contains(agentID, "repo") {
		return DefaultRepoQuestions(agentID)
	}
	if q := DefaultSpecialistQuestions(agentType); len(q) > 0 {
		return q
	}
	return DefaultRepoQuestions(agentType)
}

// DefaultSpecialistQuestions returns keyword probes per official specialist type.
func DefaultSpecialistQuestions(agentType string) []Question {
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "security":
		return []Question{
			{Prompt: "What is SQL injection and how do you prevent it?", Keywords: []string{"parameter", "sanitize", "injection", "query"}},
			{Prompt: "List three OWASP Top 10 categories.", Keywords: []string{"injection", "broken", "auth", "security"}},
		}
	case "code-review":
		return []Question{
			{Prompt: "What should you check in a pull request review?", Keywords: []string{"test", "readability", "bug", "style"}},
		}
	case "backend":
		return []Question{
			{Prompt: "Write a SQL query to list users created in the last 7 days.", Keywords: []string{"select", "where", "date", "users"}},
		}
	case "biology":
		return []Question{
			{Prompt: "What is the central dogma of molecular biology?", Keywords: []string{"dna", "rna", "protein", "transcription"}},
		}
	case "cad":
		return []Question{
			{Prompt: "What is parametric modeling in OpenSCAD?", Keywords: []string{"parameter", "variable", "model", "scad"}},
			{Prompt: "How do you export a mesh for 3D printing?", Keywords: []string{"stl", "export", "mesh"}},
		}
	case "aws":
		return []Question{
			{Prompt: "How do you list EC2 instances with the AWS CLI?", Keywords: []string{"ec2", "describe", "instances", "aws"}},
		}
	case "incident":
		return []Question{
			{Prompt: "What are the first steps in incident triage?", Keywords: []string{"severity", "impact", "communicate", "mitigate"}},
		}
	case "browser":
		return []Question{
			{Prompt: "What is accessibility testing for web apps?", Keywords: []string{"a11y", "wcag", "screen", "keyboard"}},
		}
	case "music":
		return []Question{
			{Prompt: "Suggest a verse-chorus structure for a pop song.", Keywords: []string{"verse", "chorus", "bridge", "hook"}},
		}
	default:
		return nil
	}
}
