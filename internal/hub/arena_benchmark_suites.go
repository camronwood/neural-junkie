package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
	"gopkg.in/yaml.v3"
)

const packModelArena = "model-arena"

// ArenaBenchmarkSuite is a named set of arena challenges for UI benchmarking.
type ArenaBenchmarkSuite struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Challenges  []string `json:"challenges"`
}

// ArenaBenchmarkSuitesResponse is returned by GET /api/arena/benchmark/suites.
type ArenaBenchmarkSuitesResponse struct {
	DefaultSuite string                `json:"default_suite"`
	Suites       []ArenaBenchmarkSuite `json:"suites"`
	Source       string                `json:"source"` // pack | embedded
}

type arenaEvalDoc struct {
	Version      int    `yaml:"version"`
	DefaultSuite string `yaml:"default_suite"`
	Suites       map[string]struct {
		Title       string   `yaml:"title"`
		Description string   `yaml:"description"`
		Challenges  []string `yaml:"challenges"`
	} `yaml:"suites"`
	Logic struct {
		MaxPuzzles  int     `yaml:"max_puzzles"`
		MaxSteps    int     `yaml:"max_steps"`
		PassRateMin float64 `yaml:"pass_rate_min"`
	} `yaml:"logic"`
	Connect4 struct {
		MaxSteps int `yaml:"max_steps"`
	} `yaml:"connect4"`
}

func embeddedArenaBenchmarkSuites() ArenaBenchmarkSuitesResponse {
	return ArenaBenchmarkSuitesResponse{
		DefaultSuite: "quick",
		Source:       "embedded",
		Suites: []ArenaBenchmarkSuite{
			{
				ID:          "quick",
				Title:       "Quick",
				Description: "Logic puzzle set plus a Connect Four completion smoke.",
				Challenges:  []string{"logic-set", "connect4-smoke"},
			},
			{
				ID:          "logic-only",
				Title:       "Logic only",
				Description: "Score models on the knights-and-knaves puzzle bank.",
				Challenges:  []string{"logic-set"},
			},
		},
	}
}

func arenaBenchmarkEvalPath(packDir string, m *packs.Manifest) string {
	rel := "assets/eval/arena.yaml"
	if m != nil && strings.TrimSpace(m.Assets.EvalProbes) != "" {
		rel = strings.TrimSpace(m.Assets.EvalProbes)
	}
	path, err := packs.ResolvePackRelativePath(packDir, rel)
	if err != nil {
		return filepath.Join(packDir, filepath.FromSlash(rel))
	}
	return path
}

// LoadArenaBenchmarkSuites reads suite definitions from the installed Model Arena pack.
func LoadArenaBenchmarkSuites() ArenaBenchmarkSuitesResponse {
	fallback := embeddedArenaBenchmarkSuites()
	dir, err := packs.InstalledPackDir(packModelArena)
	if err != nil || strings.TrimSpace(dir) == "" {
		return fallback
	}
	m, _ := packs.LoadManifest(dir)
	path := arenaBenchmarkEvalPath(dir, m)
	raw, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	var doc arenaEvalDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fallback
	}
	out := ArenaBenchmarkSuitesResponse{
		DefaultSuite: strings.TrimSpace(doc.DefaultSuite),
		Source:       "pack",
		Suites:       nil,
	}
	for id, meta := range doc.Suites {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		challenges := make([]string, 0, len(meta.Challenges))
		for _, c := range meta.Challenges {
			c = strings.TrimSpace(c)
			if c != "" {
				challenges = append(challenges, c)
			}
		}
		if len(challenges) == 0 {
			continue
		}
		title := strings.TrimSpace(meta.Title)
		if title == "" {
			title = id
		}
		out.Suites = append(out.Suites, ArenaBenchmarkSuite{
			ID:          id,
			Title:       title,
			Description: strings.TrimSpace(meta.Description),
			Challenges:  challenges,
		})
	}
	if len(out.Suites) == 0 {
		return fallback
	}
	// Stable order: quick first, then logic-only, then others by id.
	order := map[string]int{"quick": 0, "logic-only": 1}
	for i := 0; i < len(out.Suites); i++ {
		for j := i + 1; j < len(out.Suites); j++ {
			ai, aok := order[out.Suites[i].ID]
			bi, bok := order[out.Suites[j].ID]
			swap := false
			if aok && bok {
				swap = ai > bi
			} else if aok {
				swap = false
			} else if bok {
				swap = true
			} else {
				swap = out.Suites[i].ID > out.Suites[j].ID
			}
			if swap {
				out.Suites[i], out.Suites[j] = out.Suites[j], out.Suites[i]
			}
		}
	}
	if out.DefaultSuite == "" {
		out.DefaultSuite = out.Suites[0].ID
	}
	found := false
	for _, s := range out.Suites {
		if s.ID == out.DefaultSuite {
			found = true
			break
		}
	}
	if !found {
		out.DefaultSuite = out.Suites[0].ID
	}
	return out
}

func resolveArenaBenchmarkSuite(suiteID string) (ArenaBenchmarkSuite, error) {
	suites := LoadArenaBenchmarkSuites()
	id := strings.TrimSpace(suiteID)
	if id == "" {
		id = suites.DefaultSuite
	}
	for _, s := range suites.Suites {
		if s.ID == id {
			return s, nil
		}
	}
	return ArenaBenchmarkSuite{}, fmt.Errorf("unknown benchmark suite %q", id)
}

func arenaLogicPassRateMin() float64 {
	dir, err := packs.InstalledPackDir(packModelArena)
	if err != nil {
		return 0.5
	}
	m, _ := packs.LoadManifest(dir)
	path := arenaBenchmarkEvalPath(dir, m)
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0.5
	}
	var doc arenaEvalDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return 0.5
	}
	if doc.Logic.PassRateMin > 0 {
		return doc.Logic.PassRateMin
	}
	return 0.5
}

func arenaLogicMaxSteps() int {
	dir, err := packs.InstalledPackDir(packModelArena)
	if err != nil {
		return 5
	}
	m, _ := packs.LoadManifest(dir)
	path := arenaBenchmarkEvalPath(dir, m)
	raw, err := os.ReadFile(path)
	if err != nil {
		return 5
	}
	var doc arenaEvalDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return 5
	}
	if doc.Logic.MaxSteps > 0 {
		return doc.Logic.MaxSteps
	}
	return 5
}

func arenaConnect4MaxSteps() int {
	dir, err := packs.InstalledPackDir(packModelArena)
	if err != nil {
		return 42
	}
	m, _ := packs.LoadManifest(dir)
	path := arenaBenchmarkEvalPath(dir, m)
	raw, err := os.ReadFile(path)
	if err != nil {
		return 42
	}
	var doc arenaEvalDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return 42
	}
	if doc.Connect4.MaxSteps > 0 {
		return doc.Connect4.MaxSteps
	}
	return 42
}
