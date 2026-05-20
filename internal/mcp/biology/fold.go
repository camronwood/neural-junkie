package biology

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/mcp"
)

func esmfoldModel() string {
	return biologySettings().ESMFoldModelOrDefault()
}

func bioArtifactsDir() (string, error) {
	if d := biologySettings().ArtifactsDirOrDefault(); d != "" {
		if strings.HasPrefix(d, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			d = filepath.Join(home, d[2:])
		}
		return d, os.MkdirAll(d, 0755)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".neural-junkie", "bio")
	return dir, os.MkdirAll(dir, 0755)
}

// foldProteinSequence calls HF Inference for ESMFold and writes a PDB file.
func foldProteinSequence(ctx context.Context, raw string) (string, error) {
	seq := normalizeSequence(raw)
	if seq == "" {
		return "", fmt.Errorf("no protein sequence provided")
	}
	kind := classifySequence(seq)
	if kind != seqProtein {
		return "", fmt.Errorf("sequence does not look like protein (use analyze_sequence first)")
	}
	maxLen := maxFoldLength()
	if len(seq) > maxLen {
		return "", fmt.Errorf("sequence length %d exceeds max %d for folding", len(seq), maxLen)
	}

	token := hfhub.TokenFromConfig(mcp.AppConfig())
	if token == "" {
		return "", fmt.Errorf("Hugging Face token required for structure prediction (Settings → Hugging Face hub token or a huggingface provider)")
	}

	model := esmfoldModel()
	url := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", model)

	body, _ := json.Marshal(map[string]string{"inputs": seq})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ESMFold request failed: %w", err)
	}
	defer resp.Body.Close()

	pdbBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ESMFold API status %d: %s", resp.StatusCode, truncate(string(pdbBytes), 500))
	}
	if len(pdbBytes) < 20 || !bytes.Contains(pdbBytes, []byte("ATOM")) {
		return "", fmt.Errorf("unexpected ESMFold response (not PDB); %s", truncate(string(pdbBytes), 200))
	}

	dir, err := bioArtifactsDir()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("fold_%d.pdb", time.Now().UnixNano())
	outPath := filepath.Join(dir, name)
	if err := os.WriteFile(outPath, pdbBytes, 0644); err != nil {
		return "", fmt.Errorf("write PDB: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Structure prediction complete (in silico, ESMFold)\n")
	fmt.Fprintf(&b, "Model: %s\n", model)
	fmt.Fprintf(&b, "Sequence length: %d aa\n", len(seq))
	fmt.Fprintf(&b, "PDB file: %s\n", outPath)
	fmt.Fprintf(&b, "Open in PyMOL, ChimeraX, or similar.\n")
	fmt.Fprintf(&b, "\nThis is a computational model, not experimental structure data.\n")
	return b.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
