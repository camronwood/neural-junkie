package biology

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/scananalysis"
)

func run12PlexQCPath(path string, writeReport bool) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	dir, err := scananalysis.ResolveAnalysisDir(path)
	if err != nil {
		return "", err
	}
	doc, _, err := scananalysis.LoadAnalysis(path)
	if err != nil {
		return "", err
	}
	report, err := scananalysis.Run12PlexQC(dir, doc, scananalysis.QCOptions{
		WriteReport: writeReport,
		AnalysisDir: dir,
	})
	if err != nil {
		return "", err
	}
	return scananalysis.FormatPanelQCMarkdown(report), nil
}

func summarizePanelQCPath(path string) (string, error) {
	return run12PlexQCPath(path, false)
}

func summarizeComparatorOutputPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	sum, err := scananalysis.LoadComparatorSummary(path)
	if err != nil {
		return "", err
	}
	return scananalysis.FormatComparatorMarkdown(sum), nil
}

func runSecondaryAnalysisWorkflow(workflow, configJSON string) (string, error) {
	workflow = strings.TrimSpace(workflow)
	if workflow == "" {
		return "", fmt.Errorf("workflow is required")
	}
	configJSON = strings.TrimSpace(configJSON)
	if configJSON == "" {
		configJSON = "{}"
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "", fmt.Errorf("invalid config_json: %w", err)
	}
	switch workflow {
	case "12plex_qc":
		path, _ := cfg["analysis_dir"].(string)
		if path == "" {
			path, _ = cfg["path"].(string)
		}
		return run12PlexQCPath(path, false)
	case "summarize_comparator":
		path, _ := cfg["path"].(string)
		if path == "" {
			path, _ = cfg["analysis_dir"].(string)
		}
		return summarizeComparatorOutputPath(path)
	case "comparator", "endogenous", "std_curves", "print_order", "12plex_qc_excel", "spc_charts":
		return runSyncPythonWorkflow(workflow, cfg)
	default:
		return "", fmt.Errorf("unknown workflow %q", workflow)
	}
}

func runSyncPythonWorkflow(workflow string, cfg map[string]any) (string, error) {
	settings := biologySettings()
	toolsPath := strings.TrimSpace(settings.SecondaryAnalysisToolsPathOrDefault())
	if toolsPath == "" {
		return "", fmt.Errorf("secondary_analysis_tools_path is not configured (customer pack → Settings → Life sciences tools)")
	}
	script := map[string]string{
		"comparator":      filepath.Join("cli", "run_comparator.py"),
		"endogenous":      filepath.Join("cli", "run_endogenous.py"),
		"std_curves":      filepath.Join("cli", "run_std_curves.py"),
		"print_order":     filepath.Join("cli", "run_print_order.py"),
		"12plex_qc_excel": filepath.Join("cli", "run_12plex_qc.py"),
		"spc_charts":      filepath.Join("cli", "run_spc_charts.py"),
	}[workflow]
	if script == "" {
		return "", fmt.Errorf("unsupported workflow %q", workflow)
	}
	scriptPath := filepath.Join(toolsPath, script)
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fmt.Errorf("script not found: %s", scriptPath)
	}

	tmpDir, err := os.MkdirTemp("", "nj-mcp-sat-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	cfgPath := filepath.Join(tmpDir, "config.json")
	if cfg["out_dir"] == nil || cfg["out_dir"] == "" {
		cfg["out_dir"] = filepath.Join(tmpDir, "output")
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(cfgPath, b, 0o644); err != nil {
		return "", err
	}

	python := strings.TrimSpace(settings.PythonExecutableOrDefault())
	if python == "" {
		python = "python3"
	}
	cmd := exec.Command(python, scriptPath, "--config", cfgPath)
	cmd.Dir = toolsPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w\n%s", workflow, err, strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("## Secondary analysis: %s\n\nCompleted at %s.\n\n```json\n%s\n```\n\nLog:\n```\n%s\n```",
		workflow, time.Now().Format(time.RFC3339), strings.TrimSpace(string(b)), strings.TrimSpace(string(out))), nil
}
