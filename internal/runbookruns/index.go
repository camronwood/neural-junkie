package runbookruns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RunRecord is a lightweight index entry for a runbook execution.
type RunRecord struct {
	ID                string    `json:"id"`
	DefinitionID      string    `json:"definition_id,omitempty"`
	DefinitionVersion int       `json:"definition_version,omitempty"`
	RunNumber         int       `json:"run_number"`
	StartedAt         time.Time `json:"started_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Phase             string    `json:"phase"`
	Channel           string    `json:"channel,omitempty"`
	Title             string    `json:"title,omitempty"`
	EventLogPath      string    `json:"event_log_path,omitempty"`
	Outcome           string    `json:"outcome,omitempty"`
}

type indexFile struct {
	Runs []RunRecord `json:"runs"`
}

var mu sync.Mutex

func indexPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "runbook-runs", "index.json"), nil
}

func loadIndex() (indexFile, error) {
	path, err := indexPath()
	if err != nil {
		return indexFile{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return indexFile{Runs: []RunRecord{}}, nil
		}
		return indexFile{}, err
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return indexFile{}, err
	}
	if idx.Runs == nil {
		idx.Runs = []RunRecord{}
	}
	return idx, nil
}

func saveIndex(idx indexFile) error {
	path, err := indexPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "index-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// AppendRun adds or updates a run record.
func AppendRun(rec RunRecord) error {
	mu.Lock()
	defer mu.Unlock()
	idx, err := loadIndex()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if rec.StartedAt.IsZero() {
		rec.StartedAt = now
	}
	rec.UpdatedAt = now
	found := false
	for i := range idx.Runs {
		if idx.Runs[i].ID == rec.ID {
			idx.Runs[i] = rec
			found = true
			break
		}
	}
	if !found {
		idx.Runs = append(idx.Runs, rec)
	}
	return saveIndex(idx)
}

// ListRuns returns runs optionally filtered by definition_id.
func ListRuns(definitionID string) ([]RunRecord, error) {
	mu.Lock()
	defer mu.Unlock()
	idx, err := loadIndex()
	if err != nil {
		return nil, err
	}
	if definitionID == "" {
		out := make([]RunRecord, len(idx.Runs))
		copy(out, idx.Runs)
		return out, nil
	}
	var out []RunRecord
	for _, r := range idx.Runs {
		if r.DefinitionID == definitionID {
			out = append(out, r)
		}
	}
	return out, nil
}

// NextRunNumber returns the next run number for a definition.
func NextRunNumber(definitionID string) (int, error) {
	runs, err := ListRuns(definitionID)
	if err != nil {
		return 1, err
	}
	max := 0
	for _, r := range runs {
		if r.RunNumber > max {
			max = r.RunNumber
		}
	}
	return max + 1, nil
}

// GetRun returns one run record by collaboration id.
func GetRun(collabID string) (*RunRecord, error) {
	mu.Lock()
	defer mu.Unlock()
	idx, err := loadIndex()
	if err != nil {
		return nil, err
	}
	for i := range idx.Runs {
		if idx.Runs[i].ID == collabID {
			r := idx.Runs[i]
			return &r, nil
		}
	}
	return nil, nil
}
