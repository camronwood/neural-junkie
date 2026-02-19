package repo

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a repository for file changes
type Watcher struct {
	watcher       *fsnotify.Watcher
	repoPath      string
	onChange      func(path string)
	debounceTimer *time.Timer
	pendingChange bool
	stopCh        chan struct{}
	mu            sync.Mutex // Protects debounceTimer and pendingChange
}

// NewWatcher creates a new file system watcher for a repository
func NewWatcher(repoPath string, onChange func(path string)) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  fsWatcher,
		repoPath: repoPath,
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}

	// Add repository directory
	if err := w.addRecursive(repoPath); err != nil {
		fsWatcher.Close()
		return nil, err
	}

	return w, nil
}

// Start begins watching for file changes
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[Watcher] Error: %v", err)
			}
		}
	}()
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.stopCh)
	w.mu.Lock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.mu.Unlock()
	w.watcher.Close()
}

// handleEvent processes a file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Filter out events we don't care about
	if w.shouldIgnoreEvent(event) {
		return
	}

	w.mu.Lock()
	// Debounce rapid changes - wait 500ms after last change
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	w.pendingChange = true
	w.debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
		w.mu.Lock()
		if w.pendingChange && w.onChange != nil {
			log.Printf("[Watcher] Detected changes in %s, triggering reindex", w.repoPath)
			w.onChange(w.repoPath)
			w.pendingChange = false
		}
		w.mu.Unlock()
	})
	w.mu.Unlock()

	// If a new directory was created, add it to the watch list
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := filepath.Glob(event.Name); err == nil && len(info) > 0 {
			w.watcher.Add(event.Name)
		}
	}
}

// shouldIgnoreEvent checks if an event should be ignored
func (w *Watcher) shouldIgnoreEvent(event fsnotify.Event) bool {
	name := filepath.Base(event.Name)

	// Ignore hidden files
	if strings.HasPrefix(name, ".") && name != ".env.example" {
		return true
	}

	// Ignore temporary files
	if strings.HasSuffix(name, "~") || strings.HasSuffix(name, ".swp") {
		return true
	}

	// Check if path contains ignored directories
	pathParts := strings.Split(event.Name, string(filepath.Separator))
	for _, part := range pathParts {
		if ShouldIgnore(part) {
			return true
		}
	}

	// Ignore chmod events (we only care about writes and creates)
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return true
	}

	return false
}

// addRecursive adds all subdirectories to the watch list
func (w *Watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Skip ignored directories
		if ShouldIgnore(filepath.Base(path)) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := w.watcher.Add(path); err != nil {
			log.Printf("[Watcher] Warning: failed to watch %s: %v", path, err)
		}

		return nil
	})
}
