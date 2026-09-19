package filechange

import (
	"fmt"
	"os"
	"strings"
)

// batchPathSnapshot captures pre-batch file state for one workspace path.
// Create targets that do not exist yet are recorded with Existed=false.
type batchPathSnapshot struct {
	Content []byte
	Existed bool
}

// batchCheckpoint is a pre-apply snapshot of all member paths for one request.
type batchCheckpoint struct {
	paths map[string]batchPathSnapshot // absolute (or executor) path -> snapshot
}

// memberPaths returns every path a change may create, modify, or remove.
func memberPaths(change *FileChange) []string {
	if change == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	switch change.Operation {
	case FileOperationMove:
		add(change.OldPath)
		add(change.NewPath)
		add(change.FilePath)
	default:
		add(change.FilePath)
	}
	return out
}

func (fcm *FileChangeManager) executorForChange(changeID string) *FileChangeExecutor {
	if bound := fcm.executors[changeID]; bound != nil {
		return bound
	}
	return fcm.executor
}

// captureBatchCheckpoint snapshots all member paths for requestID before apply.
func (fcm *FileChangeManager) captureBatchCheckpoint(requestID string) error {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	req, ok := fcm.requests[requestID]
	if !ok || req == nil {
		return fmt.Errorf("file change request not found: %s", requestID)
	}

	cp := &batchCheckpoint{paths: make(map[string]batchPathSnapshot)}
	for _, change := range req.Changes {
		if change == nil {
			continue
		}
		exec := fcm.executorForChange(change.ID)
		for _, path := range memberPaths(change) {
			if _, already := cp.paths[path]; already {
				continue
			}
			snap, err := snapshotPath(exec, path)
			if err != nil {
				return fmt.Errorf("snapshot %s: %w", path, err)
			}
			cp.paths[path] = snap
		}
	}

	if fcm.checkpoints == nil {
		fcm.checkpoints = make(map[string]*batchCheckpoint)
	}
	fcm.checkpoints[requestID] = cp
	return nil
}

func snapshotPath(exec *FileChangeExecutor, path string) (batchPathSnapshot, error) {
	if exec == nil {
		return batchPathSnapshot{}, fmt.Errorf("nil executor")
	}
	_, err := exec.ioStat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return batchPathSnapshot{Existed: false}, nil
		}
		// WorkspaceIO backends may not wrap with os.IsNotExist; treat missing as absent.
		if isNotExistErr(err) {
			return batchPathSnapshot{Existed: false}, nil
		}
		return batchPathSnapshot{}, err
	}
	content, err := exec.ioRead(path)
	if err != nil {
		if isNotExistErr(err) {
			return batchPathSnapshot{Existed: false}, nil
		}
		return batchPathSnapshot{}, err
	}
	return batchPathSnapshot{
		Content: append([]byte(nil), content...),
		Existed: true,
	}, nil
}

func isNotExistErr(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}

// restoreAppliedMembers restores disk state for members that were applied before
// a mid-batch failure, using the pre-batch checkpoint.
func (fcm *FileChangeManager) restoreAppliedMembers(requestID string, appliedIDs []string) error {
	fcm.mu.Lock()
	cp := fcm.checkpoints[requestID]
	req := fcm.requests[requestID]
	idSet := make(map[string]bool, len(appliedIDs))
	for _, id := range appliedIDs {
		idSet[id] = true
	}
	type restoreJob struct {
		exec *FileChangeExecutor
		path string
		snap batchPathSnapshot
	}
	var jobs []restoreJob
	if cp != nil && req != nil {
		// Reverse order so later moves/edits unwind first.
		for i := len(req.Changes) - 1; i >= 0; i-- {
			change := req.Changes[i]
			if change == nil || !idSet[change.ID] {
				continue
			}
			exec := fcm.executorForChange(change.ID)
			paths := memberPaths(change)
			for j := len(paths) - 1; j >= 0; j-- {
				path := paths[j]
				snap, ok := cp.paths[path]
				if !ok {
					continue
				}
				jobs = append(jobs, restoreJob{exec: exec, path: path, snap: snap})
			}
		}
	}
	fcm.mu.Unlock()

	var firstErr error
	for _, job := range jobs {
		if err := restorePath(job.exec, job.path, job.snap); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("restore %s: %w", job.path, err)
		}
	}
	return firstErr
}

func restorePath(exec *FileChangeExecutor, path string, snap batchPathSnapshot) error {
	if exec == nil {
		return fmt.Errorf("nil executor")
	}
	if !snap.Existed {
		err := exec.ioRemove(path)
		if err != nil && !isNotExistErr(err) {
			return err
		}
		return nil
	}
	return exec.ioWrite(path, snap.Content)
}

func (fcm *FileChangeManager) clearBatchCheckpoint(requestID string) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	delete(fcm.checkpoints, requestID)
}
