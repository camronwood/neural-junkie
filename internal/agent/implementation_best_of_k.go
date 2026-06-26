package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	defaultBootFixBestOfK = 2
	maxBestOfK            = 5
)

type workspaceSnapshot struct {
	files map[string][]byte
}

func implementationBestOfK(msg *protocol.Message) int {
	if msg == nil || msg.Metadata == nil {
		return 1
	}
	if v, ok := msg.Metadata["implementation_best_of_k"].(float64); ok && int(v) > 1 {
		return clampBestOfK(int(v))
	}
	if v, ok := msg.Metadata["implementation_best_of_k"].(int); ok && v > 1 {
		return clampBestOfK(v)
	}
	if bootFixBestOfKEnabled(msg) {
		return defaultBootFixBestOfK
	}
	return 1
}

func bootFixBestOfKEnabled(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	if v, ok := msg.Metadata["implementation_best_of_k_boot_fix"].(bool); ok && v {
		return true
	}
	return false
}

func clampBestOfK(k int) int {
	if k < 2 {
		return 1
	}
	if k > maxBestOfK {
		return maxBestOfK
	}
	return k
}

func snapshotWorkspaceFiles(wsPath string, relPaths []string) (*workspaceSnapshot, error) {
	snap := &workspaceSnapshot{files: make(map[string][]byte, len(relPaths))}
	for _, rel := range relPaths {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		full := filepath.Join(wsPath, rel)
		data, err := os.ReadFile(full)
		if err != nil {
			if os.IsNotExist(err) {
				snap.files[rel] = nil
				continue
			}
			return nil, err
		}
		snap.files[rel] = append([]byte(nil), data...)
	}
	return snap, nil
}

func restoreWorkspaceFiles(wsPath string, snap *workspaceSnapshot) error {
	if snap == nil {
		return nil
	}
	for rel, data := range snap.files {
		full := filepath.Join(wsPath, rel)
		if data == nil {
			_ = os.Remove(full)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func workspaceRetrySnapshotPaths(wsPath string, manifest *StackManifest) []string {
	seen := make(map[string]bool)
	var paths []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	for _, p := range bootFixBootstrapReadPaths(wsPath, manifest) {
		add(p)
	}
	if manifest != nil {
		if manifest.TailwindConfig != "" {
			add(manifest.TailwindConfig)
		}
		if manifest.EntryPoint != "" {
			add(manifest.EntryPoint)
		}
	}
	for _, name := range []string{"Makefile", "package.json", "src/App.tsx", "src/App.jsx", "src/main.tsx", "src/main.jsx"} {
		if fileExists(filepath.Join(wsPath, name)) {
			add(name)
		}
	}
	return paths
}

type implSessionRunResult struct {
	text       string
	streamID   string
	proposed   bool
	files      []string
	outcome    map[string]interface{}
	score      int
}

func (a *Agent) runImplementationSessionBestOfK(
	ctx context.Context,
	msg *protocol.Message,
	eff ai.AIProvider,
	streamMsgID string,
) (string, string, bool, []string, map[string]interface{}, error) {
	k := implementationBestOfK(msg)
	if k <= 1 {
		return a.runImplementationSessionStreaming(ctx, msg, eff, streamMsgID)
	}

	wsPath := a.resolveWorkspacePath(msg)
	var manifest *StackManifest
	if wsPath != "" {
		manifest = DetectStackManifest(wsPath)
	}
	snap, snapErr := snapshotWorkspaceFiles(wsPath, workspaceRetrySnapshotPaths(wsPath, manifest))

	var best *implSessionRunResult
	for run := 1; run <= k; run++ {
		if run > 1 && snapErr == nil && snap != nil {
			_ = restoreWorkspaceFiles(wsPath, snap)
		}
		text, sid, proposed, files, outcome, err := a.runImplementationSessionStreaming(ctx, msg, eff, streamMsgID)
		if err != nil {
			if best != nil {
				return best.text, best.streamID, best.proposed, best.files, best.outcome, nil
			}
			return text, sid, proposed, files, outcome, err
		}
		score := outcomeScore(outcome)
		if outcome == nil {
			outcome = map[string]interface{}{}
		}
		outcome["best_of_k_total"] = k
		outcome["best_of_k_run"] = run
		outcome["best_of_k_selected"] = false
		candidate := &implSessionRunResult{
			text: text, streamID: sid, proposed: proposed, files: files, outcome: outcome, score: score,
		}
		if best == nil || candidate.score > best.score {
			best = candidate
		}
		if score >= 100 {
			break
		}
	}

	if best == nil {
		return a.runImplementationSessionStreaming(ctx, msg, eff, streamMsgID)
	}
	if best.outcome != nil {
		best.outcome["best_of_k_selected"] = true
		if run, ok := best.outcome["best_of_k_run"].(int); ok {
			if note := formatBestOfKOutcomeNote(run, k); note != "" {
				best.text = strings.TrimSpace(best.text + "\n\n" + note)
			}
		}
	}
	return best.text, best.streamID, best.proposed, best.files, best.outcome, nil
}
