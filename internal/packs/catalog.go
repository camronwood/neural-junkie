package packs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// DefaultCatalogURL is the official pack store listing on GitHub (main branch).
const DefaultCatalogURL = "https://raw.githubusercontent.com/camronwood/neural-junkie/main/packs/catalog.json"

var (
	catalogClient   = &http.Client{Timeout: 30 * time.Second}
	catalogCacheMu  sync.RWMutex
	catalogCache    *Catalog
	catalogCacheAt  time.Time
	catalogCacheTTL = 5 * time.Minute
)

// CatalogURL returns the remote catalog JSON URL (env override).
func CatalogURL() string {
	if u := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_PACKS_CATALOG_URL")); u != "" {
		return u
	}
	return DefaultCatalogURL
}

// FetchCatalog loads the remote pack store catalog.
func FetchCatalog() (*Catalog, error) {
	catalogCacheMu.RLock()
	if catalogCache != nil && time.Since(catalogCacheAt) < catalogCacheTTL {
		c := cloneCatalog(catalogCache)
		catalogCacheMu.RUnlock()
		return c, nil
	}
	catalogCacheMu.RUnlock()

	cat, err := fetchRemoteCatalog(CatalogURL())
	if err != nil {
		return nil, fmt.Errorf("pack catalog: %w", err)
	}
	cat = orderCatalogPacks(cat)

	catalogCacheMu.Lock()
	catalogCache = cat
	catalogCacheAt = time.Now()
	catalogCacheMu.Unlock()

	return cloneCatalog(cat), nil
}

// InvalidateCatalogCache clears the in-memory catalog cache (tests).
func InvalidateCatalogCache() {
	catalogCacheMu.Lock()
	catalogCache = nil
	catalogCacheAt = time.Time{}
	catalogCacheMu.Unlock()
}

func fetchRemoteCatalog(url string) (*Catalog, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := catalogClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GET %s: %s (%s)", url, resp.Status, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &cat, nil
}

func orderCatalogPacks(cat *Catalog) *Catalog {
	if cat == nil {
		return &Catalog{Version: 1, Packs: nil}
	}
	byID := make(map[string]CatalogEntry, len(cat.Packs))
	for _, e := range cat.Packs {
		id := strings.TrimSpace(e.ID)
		if id == "" {
			continue
		}
		byID[id] = e
	}
	out := &Catalog{Version: cat.Version}
	if out.Version == 0 {
		out.Version = 1
	}
	for _, id := range OfficialPackIDs {
		if e, ok := byID[id]; ok {
			out.Packs = append(out.Packs, e)
			delete(byID, id)
		}
	}
	for _, e := range byID {
		out.Packs = append(out.Packs, e)
	}
	return out
}

func cloneCatalog(c *Catalog) *Catalog {
	if c == nil {
		return nil
	}
	cp := &Catalog{Version: c.Version, Packs: make([]CatalogEntry, len(c.Packs))}
	copy(cp.Packs, c.Packs)
	return cp
}

// CatalogEntryByID returns a catalog row or nil.
func (c *Catalog) CatalogEntryByID(packID string) *CatalogEntry {
	if c == nil {
		return nil
	}
	packID = strings.TrimSpace(packID)
	for i := range c.Packs {
		if c.Packs[i].ID == packID {
			cp := c.Packs[i]
			return &cp
		}
	}
	return nil
}
