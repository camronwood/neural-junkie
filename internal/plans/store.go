package plans

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Record is a persisted plan file plus parsed fields.
type Record struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Overview  string    `json:"overview"`
	Todos     []Todo    `json:"todos"`
	Markdown  string    `json:"markdown"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store persists *.plan.md files under a directory.
type Store struct {
	dir string
	mu  sync.Mutex
}

var (
	defaultOnce   sync.Once
	defaultStore  *Store
	overrideStore *Store
	slugCleanRE   = regexp.MustCompile(`[^a-z0-9]+`)
	idSafeRE      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,180}$`)
)

// Default returns the process store under ~/.neural-junkie/plans.
func Default() *Store {
	defaultOnce.Do(func() {
		dir := filepath.Join(os.Getenv("HOME"), ".neural-junkie", "plans")
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			dir = filepath.Join(home, ".neural-junkie", "plans")
		}
		defaultStore = NewStore(dir)
	})
	return defaultStore
}

// Active is Default unless a test override is set.
func Active() *Store {
	if overrideStore != nil {
		return overrideStore
	}
	return Default()
}

// OverrideForTest swaps the HTTP/process store. Restore with the returned func.
func OverrideForTest(s *Store) func() {
	prev := overrideStore
	overrideStore = s
	return func() { overrideStore = prev }
}

// NewStore creates a store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Dir is the on-disk plans folder.
func (s *Store) Dir() string {
	if s == nil {
		return ""
	}
	return s.dir
}

// SaveFromMarkdown parses content and writes a new plan file. Returns nil, nil if not a plan.
func (s *Store) SaveFromMarkdown(content string) (*Record, error) {
	doc, err := Parse(content)
	if err != nil {
		return nil, nil
	}
	id := newPlanID(doc.Name)
	return s.write(id, Render(doc))
}

// Put replaces markdown for an existing id (or creates it).
func (s *Store) Put(id, markdown string) (*Record, error) {
	id = strings.TrimSpace(id)
	if !idSafeRE.MatchString(id) {
		return nil, fmt.Errorf("invalid plan id")
	}
	doc, err := Parse(markdown)
	if err != nil {
		return nil, err
	}
	return s.write(id, Render(doc))
}

// Get loads a plan by id.
func (s *Store) Get(id string) (*Record, error) {
	id = strings.TrimSpace(id)
	if !idSafeRE.MatchString(id) {
		return nil, fmt.Errorf("invalid plan id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, id+".plan.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := Parse(string(raw))
	if err != nil {
		return nil, err
	}
	info, _ := os.Stat(path)
	rec := &Record{
		ID:       id,
		Name:     doc.Name,
		Overview: doc.Overview,
		Todos:    doc.Todos,
		Markdown: string(raw),
		Path:     path,
	}
	if info != nil {
		rec.UpdatedAt = info.ModTime()
	}
	return rec, nil
}

// List returns plans newest first.
func (s *Store) List() ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".plan.md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".plan.md")
		path := filepath.Join(s.dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		doc, err := Parse(string(raw))
		if err != nil {
			continue
		}
		rec := Record{
			ID:       id,
			Name:     doc.Name,
			Overview: doc.Overview,
			Todos:    doc.Todos,
			Markdown: string(raw),
			Path:     path,
		}
		if info, err := e.Info(); err == nil {
			rec.UpdatedAt = info.ModTime()
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (s *Store) write(id, markdown string) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, id+".plan.md")
	if err := os.WriteFile(path, []byte(markdown), 0o644); err != nil {
		return nil, err
	}
	doc, err := Parse(markdown)
	if err != nil {
		return nil, err
	}
	return &Record{
		ID:        id,
		Name:      doc.Name,
		Overview:  doc.Overview,
		Todos:     doc.Todos,
		Markdown:  markdown,
		Path:      path,
		UpdatedAt: time.Now(),
	}, nil
}

// Render rebuilds Cursor-shaped markdown from a parsed document.
func Render(doc Document) string {
	if strings.TrimSpace(doc.Raw) != "" {
		if _, err := Parse(doc.Raw); err == nil {
			return strings.TrimSpace(doc.Raw) + "\n"
		}
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", yamlQuote(doc.Name))
	fmt.Fprintf(&b, "overview: %s\n", yamlQuote(doc.Overview))
	b.WriteString("todos:\n")
	for _, t := range doc.Todos {
		fmt.Fprintf(&b, "  - id: %s\n", yamlQuote(t.ID))
		fmt.Fprintf(&b, "    content: %s\n", yamlQuote(t.Content))
		status := strings.TrimSpace(t.Status)
		if status == "" {
			status = "pending"
		}
		fmt.Fprintf(&b, "    status: %s\n", status)
	}
	b.WriteString("isProject: false\n")
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(doc.Body))
	b.WriteString("\n")
	return b.String()
}

func yamlQuote(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#{}[]&*!|>'\"%@`\n") {
		escaped := strings.ReplaceAll(s, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

func newPlanID(name string) string {
	slug := slugify(name)
	if slug == "" {
		slug = "plan"
	}
	short := strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(short) > 8 {
		short = short[:8]
	}
	return slug + "_" + short
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugCleanRE.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 48 {
		s = s[:48]
		s = strings.Trim(s, "_")
	}
	return s
}
