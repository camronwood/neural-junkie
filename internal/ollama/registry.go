package ollama

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ollamaWebOrigin = "https://ollama.com"

var (
	libraryLinkRe    = regexp.MustCompile(`href="/library/([a-zA-Z0-9._-]+)"`)
	xNamespaceLinkRe = regexp.MustCompile(`href="/(x/[^":]+)"`)
	tagLinkRe        = regexp.MustCompile(`href="/library/([a-zA-Z0-9._:-]+)"`)
	xTagLinkRe       = regexp.MustCompile(`href="/(x/[^"]+)"`)
	// ollama.com tags/model pages: `llama3.1:latest</p> ... <p class="flex text-neutral-500">4.9GB ·`
	tagSizeRe = regexp.MustCompile(`(?is):([a-z0-9._-]+)</[^>]+>\s*</span>\s*<p[^>]*flex text-neutral-500[^>]*>\s*([\d.]+)\s*([GM]B)`)
	anySizeRe = regexp.MustCompile(`(?i)flex text-neutral-500[^>]*>\s*([\d.]+)\s*([GM]B)`)
)

// RegistryModel is one model family from the public Ollama library.
type RegistryModel struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// RegistryTag is one pull tag for a model family.
type RegistryTag struct {
	Name     string `json:"name"`
	SizeHint string `json:"size_hint,omitempty"`
}

// RegistrySearchResult is returned by library search.
type RegistrySearchResult struct {
	Query   string          `json:"query"`
	Page    int             `json:"page"`
	Models  []RegistryModel `json:"models"`
	HasMore bool            `json:"has_more"`
}

// RegistryTagsResult lists installable tags for one model family.
type RegistryTagsResult struct {
	Name         string        `json:"name"`
	DefaultTag   string        `json:"default_tag,omitempty"`
	SizeHint     string        `json:"size_hint,omitempty"`
	Tags         []RegistryTag `json:"tags"`
	AllTagsCount int           `json:"all_tags_count,omitempty"`
}

type registryClient struct {
	httpClient  *http.Client
	cacheMu     sync.RWMutex
	searchCache map[string]cachedSearch
	tagsCache   map[string]cachedTags
}

type cachedSearch struct {
	at     time.Time
	result RegistrySearchResult
}

type cachedTags struct {
	at     time.Time
	result RegistryTagsResult
}

var defaultRegistry = &registryClient{
	httpClient:  &http.Client{Timeout: 20 * time.Second},
	searchCache: make(map[string]cachedSearch),
	tagsCache:   make(map[string]cachedTags),
}

const registryCacheTTL = 5 * time.Minute

// SearchRegistry queries ollama.com for model families matching q.
func SearchRegistry(ctx context.Context, query string, page int) (RegistrySearchResult, error) {
	return defaultRegistry.search(ctx, query, page)
}

// ListRegistryTags returns common pull tags for a model family.
func ListRegistryTags(ctx context.Context, name string) (RegistryTagsResult, error) {
	return defaultRegistry.tags(ctx, name)
}

func (c *registryClient) search(ctx context.Context, query string, page int) (RegistrySearchResult, error) {
	query = strings.TrimSpace(query)
	if page < 1 {
		page = 1
	}
	cacheKey := query + "\x00" + strconv.Itoa(page)
	c.cacheMu.RLock()
	if hit, ok := c.searchCache[cacheKey]; ok && time.Since(hit.at) < registryCacheTTL {
		c.cacheMu.RUnlock()
		return hit.result, nil
	}
	c.cacheMu.RUnlock()

	target := ollamaWebOrigin + "/library"
	if query != "" {
		target = ollamaWebOrigin + "/search?" + url.Values{"q": {query}}.Encode()
	}

	body, err := c.fetch(ctx, target)
	if err != nil {
		return RegistrySearchResult{}, err
	}

	names := uniqueStrings(parseLibraryNames(body))
	const pageSize = 48
	start := (page - 1) * pageSize
	if start >= len(names) {
		return RegistrySearchResult{
			Query:   query,
			Page:    page,
			Models:  nil,
			HasMore: false,
		}, nil
	}
	end := start + pageSize
	if end > len(names) {
		end = len(names)
	}
	pageNames := names[start:end]

	out := RegistrySearchResult{
		Query:   query,
		Page:    page,
		Models:  make([]RegistryModel, 0, len(pageNames)),
		HasMore: end < len(names),
	}
	for _, name := range pageNames {
		out.Models = append(out.Models, RegistryModel{
			Name:  name,
			Title: humanizeModelName(strings.TrimPrefix(name, "x/")),
			URL:   registryModelPageURL(name),
		})
	}

	c.cacheMu.Lock()
	c.searchCache[cacheKey] = cachedSearch{at: time.Now(), result: out}
	c.cacheMu.Unlock()
	return out, nil
}

func (c *registryClient) tags(ctx context.Context, name string) (RegistryTagsResult, error) {
	name = normalizeRegistryName(name)
	if name == "" {
		return RegistryTagsResult{}, fmt.Errorf("model name is required")
	}
	c.cacheMu.RLock()
	if hit, ok := c.tagsCache[name]; ok && time.Since(hit.at) < registryCacheTTL {
		c.cacheMu.RUnlock()
		return hit.result, nil
	}
	c.cacheMu.RUnlock()

	target := registryTagsPageURL(name)
	body, err := c.fetch(ctx, target)
	if err != nil {
		return RegistryTagsResult{}, err
	}

	all := uniqueStrings(parseTagNames(body, name))
	filtered := filterCommonTags(name, all)
	defaultTag := name + ":latest"
	for _, tag := range filtered {
		if strings.HasSuffix(tag, ":latest") {
			defaultTag = tag
			break
		}
	}
	if len(filtered) == 0 && len(all) > 0 {
		filtered = all
		if len(filtered) > 24 {
			filtered = filtered[:24]
		}
		defaultTag = filtered[0]
	}

	sizeBySuffix := parseTagSizeHints(body)
	out := RegistryTagsResult{
		Name:         name,
		DefaultTag:   defaultTag,
		Tags:         make([]RegistryTag, 0, len(filtered)),
		AllTagsCount: len(all),
	}
	if hint := sizeBySuffix["latest"]; hint != "" {
		out.SizeHint = hint
	} else if hint := sizeHintForTag(defaultTag, sizeBySuffix); hint != "" {
		out.SizeHint = hint
	} else if hint := parseFirstSizeHint(body); hint != "" {
		out.SizeHint = hint
	}
	for _, tag := range filtered {
		out.Tags = append(out.Tags, RegistryTag{
			Name:     tag,
			SizeHint: sizeHintForTag(tag, sizeBySuffix),
		})
	}

	c.cacheMu.Lock()
	c.tagsCache[name] = cachedTags{at: time.Now(), result: out}
	c.cacheMu.Unlock()
	return out, nil
}

func sizeHintForTag(tag string, bySuffix map[string]string) string {
	if i := strings.LastIndex(tag, ":"); i >= 0 {
		if hint := bySuffix[tag[i+1:]]; hint != "" {
			return hint
		}
	}
	return bySuffix[tag]
}

func parseTagSizeHints(html string) map[string]string {
	out := make(map[string]string)
	for _, m := range tagSizeRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 4 {
			continue
		}
		suffix := strings.ToLower(strings.TrimSpace(m[1]))
		hint := formatScrapedSize(m[2], m[3])
		if suffix == "" || hint == "" {
			continue
		}
		if _, exists := out[suffix]; !exists {
			out[suffix] = hint
		}
	}
	return out
}

func parseFirstSizeHint(html string) string {
	if m := anySizeRe.FindStringSubmatch(html); len(m) >= 3 {
		return formatScrapedSize(m[1], m[2])
	}
	return ""
}

func formatScrapedSize(num, unit string) string {
	num = strings.TrimSpace(num)
	unit = strings.ToUpper(strings.TrimSpace(unit))
	if num == "" || (unit != "GB" && unit != "MB" && unit != "TB") {
		return ""
	}
	return "~" + num + " " + unit
}

func (c *registryClient) fetch(ctx context.Context, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Neural-Junkie/1.0 (+https://github.com/camronwood/neural-junkie)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama registry fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama registry fetch: status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseLibraryNames(html string) []string {
	var out []string
	for _, m := range libraryLinkRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" || strings.Contains(name, ":") {
			continue
		}
		out = append(out, name)
	}
	for _, m := range xNamespaceLinkRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" || strings.Contains(name, ":") {
			continue
		}
		out = append(out, name)
	}
	return out
}

func parseTagNames(html, family string) []string {
	var out []string
	for _, m := range tagLinkRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		tag := strings.TrimSpace(m[1])
		if tag == "" || !strings.HasPrefix(tag, family+":") {
			continue
		}
		out = append(out, tag)
	}
	for _, m := range xTagLinkRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		tag := strings.TrimSpace(m[1])
		if tag == "" || !strings.HasPrefix(tag, family+":") {
			continue
		}
		out = append(out, tag)
	}
	return out
}

// registryTagsPageURL returns the ollama.com tags page for a model family.
func registryTagsPageURL(name string) string {
	if strings.HasPrefix(name, "x/") {
		return fmt.Sprintf("%s/%s/tags", ollamaWebOrigin, name)
	}
	return fmt.Sprintf("%s/library/%s/tags", ollamaWebOrigin, url.PathEscape(name))
}

func registryModelPageURL(name string) string {
	if strings.HasPrefix(name, "x/") {
		return fmt.Sprintf("%s/%s", ollamaWebOrigin, name)
	}
	return fmt.Sprintf("%s/library/%s", ollamaWebOrigin, name)
}

func filterCommonTags(family string, tags []string) []string {
	var out []string
	for _, tag := range tags {
		suffix := strings.TrimPrefix(tag, family+":")
		if suffix == "latest" {
			out = append(out, tag)
			continue
		}
		if strings.Contains(suffix, "-") {
			continue
		}
		out = append(out, tag)
	}
	return out
}

func normalizeRegistryName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, ollamaWebOrigin+"/")
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimPrefix(name, "library/")
	if i := strings.Index(name, ":"); i >= 0 {
		name = name[:i]
	}
	return name
}

func humanizeModelName(name string) string {
	parts := strings.Split(name, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// MergeCatalogWithRegistry overlays curated metadata onto registry rows and prepends featured entries.
func MergeCatalogWithRegistry(curated []LibraryModel, registry []RegistryModel) []LibraryModel {
	out := append([]LibraryModel(nil), curated...)
	seenFamilies := make(map[string]struct{}, len(curated))
	seenNames := make(map[string]struct{}, len(curated)+len(registry))
	for _, row := range curated {
		seenNames[row.Name] = struct{}{}
		family := normalizeRegistryName(row.Name)
		if family != "" {
			seenFamilies[family] = struct{}{}
		}
	}

	for _, reg := range registry {
		base := normalizeRegistryName(reg.Name)
		if base == "" {
			continue
		}
		if _, ok := seenFamilies[base]; ok {
			continue
		}
		if _, ok := seenNames[base]; ok {
			continue
		}
		seenFamilies[base] = struct{}{}
		seenNames[base] = struct{}{}
		out = append(out, LibraryModel{
			Name:        base,
			Title:       reg.Title,
			Description: reg.Description,
		})
	}
	return out
}
