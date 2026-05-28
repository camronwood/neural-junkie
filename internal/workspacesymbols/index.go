package workspacesymbols

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/workspacefiles"
)

type indexCache struct {
	mu      sync.RWMutex
	entries map[string]cachedIndex
}

type cachedIndex struct {
	root    string
	mtime   time.Time
	symbols []Symbol
}

var globalIndex indexCache = indexCache{entries: make(map[string]cachedIndex)}

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".neural-junkie", "symbol-index")
	return dir, os.MkdirAll(dir, 0o755)
}

func workspaceKey(root string) string {
	h := sha256.Sum256([]byte(filepath.Clean(root)))
	return hex.EncodeToString(h[:16])
}

func diskCachePath(key string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key+".json"), nil
}

type diskIndex struct {
	Root      string    `json:"root"`
	BuiltAt   time.Time `json:"built_at"`
	RootMtime int64     `json:"root_mtime_ns"`
	Symbols   []Symbol  `json:"symbols"`
}

func rootMtime(root string) (time.Time, error) {
	st, err := os.Stat(root)
	if err != nil {
		return time.Time{}, err
	}
	return st.ModTime(), nil
}

// BuildIndex scans workspace and returns all symbols (cached on disk + memory).
func BuildIndex(ctx context.Context, workspaceRoot string) ([]Symbol, error) {
	root, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return nil, err
	}
	key := workspaceKey(root)
	mod, err := rootMtime(root)
	if err != nil {
		return nil, err
	}

	globalIndex.mu.RLock()
	if c, ok := globalIndex.entries[key]; ok && c.root == root && c.mtime.Equal(mod) {
		out := append([]Symbol(nil), c.symbols...)
		globalIndex.mu.RUnlock()
		return out, nil
	}
	globalIndex.mu.RUnlock()

	if path, err := diskCachePath(key); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			var di diskIndex
			if json.Unmarshal(data, &di) == nil && di.Root == root && di.RootMtime == mod.UnixNano() {
				globalIndex.mu.Lock()
				globalIndex.entries[key] = cachedIndex{root: root, mtime: mod, symbols: di.Symbols}
				globalIndex.mu.Unlock()
				return di.Symbols, nil
			}
		}
	}

	paths, err := workspacefiles.Search(ctx, root, "", 8000)
	if err != nil {
		return nil, err
	}
	var all []Symbol
	for _, rel := range paths {
		if ctx.Err() != nil {
			break
		}
		lang := langForPath(rel)
		if lang == "" {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		syms, err := scanFile(full, rel, lang)
		if err != nil {
			continue
		}
		all = append(all, syms...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Path != all[j].Path {
			return all[i].Path < all[j].Path
		}
		return all[i].Line < all[j].Line
	})

	globalIndex.mu.Lock()
	globalIndex.entries[key] = cachedIndex{root: root, mtime: mod, symbols: all}
	globalIndex.mu.Unlock()

	if path, err := diskCachePath(key); err == nil {
		di := diskIndex{Root: root, BuiltAt: time.Now(), RootMtime: mod.UnixNano(), Symbols: all}
		if b, err := json.Marshal(di); err == nil {
			_ = os.WriteFile(path, b, 0o644)
		}
	}
	return all, nil
}

// SearchIndexed uses BuildIndex then filters by query and kind.
func SearchIndexed(ctx context.Context, workspaceRoot, q, kind string, limit int) ([]Symbol, error) {
	if limit <= 0 {
		limit = 50
	}
	all, err := BuildIndex(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	kind = strings.ToLower(strings.TrimSpace(kind))
	var out []Symbol
	for _, s := range all {
		if kind != "" && strings.ToLower(s.Kind) != kind {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(s.Name), q) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
