package artifacts

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMaxPayloadBytes int64 = 8 << 20
	DefaultMaxAssetBytes   int64 = 64 << 20
)

var (
	ErrNotFound     = errors.New("artifact not found")
	ErrConflict     = errors.New("artifact revision conflict")
	ErrCorrupt      = errors.New("artifact data is corrupt")
	ErrTooLarge     = errors.New("artifact content exceeds size limit")
	ErrInvalidID    = errors.New("invalid artifact ID")
	ErrInvalidAsset = errors.New("invalid asset name")

	safeID    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	safeAsset = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,255}$`)
)

// Store owns artifacts and their assets beneath Root.
type Store struct {
	root       string
	maxPayload int64
	maxAsset   int64
	mu         sync.RWMutex
}

// Option configures a Store.
type Option func(*Store) error

func WithMaxPayloadBytes(n int64) Option {
	return func(s *Store) error {
		if n <= 0 {
			return fmt.Errorf("max payload bytes must be positive")
		}
		s.maxPayload = n
		return nil
	}
}

func WithMaxAssetBytes(n int64) Option {
	return func(s *Store) error {
		if n <= 0 {
			return fmt.Errorf("max asset bytes must be positive")
		}
		s.maxAsset = n
		return nil
	}
}

// DefaultRoot resolves the app-managed artifact directory without creating it.
func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neural-junkie", "artifacts"), nil
}

// NewStore opens a store rooted at root. An empty root uses DefaultRoot.
func NewStore(root string, options ...Option) (*Store, error) {
	if root == "" {
		var err error
		root, err = DefaultRoot()
		if err != nil {
			return nil, err
		}
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve artifact root: %w", err)
	}
	s := &Store{root: filepath.Clean(absolute), maxPayload: DefaultMaxPayloadBytes, maxAsset: DefaultMaxAssetBytes}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(s); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return nil, fmt.Errorf("create artifact root: %w", err)
	}
	return s, nil
}

func (s *Store) Root() string { return s.root }

// Create stores an artifact at revision one. A blank ID is generated.
func (s *Store) Create(artifact Artifact) (*Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if artifact.ID == "" {
		artifact.ID = newID()
	}
	if err := validateID(artifact.ID); err != nil {
		return nil, err
	}
	dir := s.artifactDir(artifact.ID)
	if _, err := os.Lstat(dir); err == nil {
		return nil, fmt.Errorf("%w: %s already exists", ErrConflict, artifact.ID)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("inspect artifact: %w", err)
	}
	if err := s.prepare(&artifact); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	artifact.SchemaVersion = CurrentSchemaVersion
	artifact.Revision = 1
	artifact.CreatedAt = now
	artifact.UpdatedAt = now

	if err := os.MkdirAll(filepath.Join(dir, "revisions"), 0o700); err != nil {
		return nil, fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("create asset directory: %w", err)
	}
	if err := s.writeRevision(&artifact); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if err := atomicJSON(filepath.Join(dir, "artifact.json"), &artifact); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return cloneArtifact(&artifact), nil
}

// Get returns the current artifact.
func (s *Store) Get(id string) (*Artifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getUnlocked(id)
}

// Update writes a new immutable revision when expectedRevision is current.
func (s *Store) Update(artifact Artifact, expectedRevision uint64) (*Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateID(artifact.ID); err != nil {
		return nil, err
	}
	current, err := s.getUnlocked(artifact.ID)
	if err != nil {
		return nil, err
	}
	if current.Revision != expectedRevision {
		return nil, fmt.Errorf("%w: expected %d, current %d", ErrConflict, expectedRevision, current.Revision)
	}
	if err := s.prepare(&artifact); err != nil {
		return nil, err
	}
	artifact.SchemaVersion = CurrentSchemaVersion
	artifact.Revision = current.Revision + 1
	artifact.CreatedAt = current.CreatedAt
	artifact.UpdatedAt = time.Now().UTC()
	if err := s.writeRevision(&artifact); err != nil {
		return nil, err
	}
	if err := atomicJSON(filepath.Join(s.artifactDir(artifact.ID), "artifact.json"), &artifact); err != nil {
		return nil, err
	}
	return cloneArtifact(&artifact), nil
}

// Delete removes an artifact when expectedRevision is current.
func (s *Store) Delete(id string, expectedRevision uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.getUnlocked(id)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return fmt.Errorf("%w: expected %d, current %d", ErrConflict, expectedRevision, current.Revision)
	}
	if err := os.RemoveAll(s.artifactDir(id)); err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	return syncDir(s.root)
}

// List returns current artifacts matching filter, sorted by ID.
func (s *Store) List(filter Filter) ([]Artifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	result := make([]Artifact, 0)
	for _, entry := range entries {
		if !entry.IsDir() || validateID(entry.Name()) != nil {
			continue
		}
		artifact, err := s.getUnlocked(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("list artifact %q: %w", entry.Name(), err)
		}
		if matches(*artifact, filter) {
			result = append(result, *artifact)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// GetRevision returns an immutable artifact snapshot.
func (s *Store) GetRevision(id string, revision uint64) (*ArtifactRevision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := validateID(id); err != nil {
		return nil, err
	}
	if revision == 0 {
		return nil, fmt.Errorf("%w: revision must be positive", ErrNotFound)
	}
	if err := s.ensureArtifactDir(id); err != nil {
		return nil, err
	}
	if err := s.ensureManagedDir(id, "revisions"); err != nil {
		return nil, err
	}
	var snapshot ArtifactRevision
	path := filepath.Join(s.artifactDir(id), "revisions", fmt.Sprintf("%020d.json", revision))
	if err := readJSON(path, &snapshot, s.documentLimit()); err != nil {
		return nil, mapReadError(err, id)
	}
	if snapshot.ArtifactID != id || snapshot.Revision != revision ||
		snapshot.Artifact.ID != id || snapshot.Artifact.Revision != revision {
		return nil, fmt.Errorf("%w: revision identity mismatch", ErrCorrupt)
	}
	return &snapshot, nil
}

// ListRevisions lists revision metadata in ascending revision order.
func (s *Store) ListRevisions(id string) ([]ArtifactRevision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := s.ensureArtifactDir(id); err != nil {
		return nil, err
	}
	if err := s.ensureManagedDir(id, "revisions"); err != nil {
		return nil, err
	}
	dir := filepath.Join(s.artifactDir(id), "revisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, mapReadError(err, id)
	}
	out := make([]ArtifactRevision, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		value := strings.TrimSuffix(entry.Name(), ".json")
		revision, err := strconv.ParseUint(value, 10, 64)
		if err != nil || revision == 0 {
			return nil, fmt.Errorf("%w: invalid revision filename %q", ErrCorrupt, entry.Name())
		}
		var snapshot ArtifactRevision
		if err := readJSON(filepath.Join(dir, entry.Name()), &snapshot, s.documentLimit()); err != nil {
			return nil, mapReadError(err, id)
		}
		if snapshot.ArtifactID != id || snapshot.Revision != revision ||
			snapshot.Artifact.ID != id || snapshot.Artifact.Revision != revision {
			return nil, fmt.Errorf("%w: revision identity mismatch", ErrCorrupt)
		}
		out = append(out, snapshot)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision < out[j].Revision })
	return out, nil
}

// PutAsset atomically stores a flat, safely named asset.
func (s *Store) PutAsset(id, name string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if int64(len(data)) > s.maxAsset {
		return ErrTooLarge
	}
	if err := validateID(id); err != nil {
		return err
	}
	if err := validateAssetName(name); err != nil {
		return err
	}
	if err := s.ensureArtifactDir(id); err != nil {
		return err
	}
	if err := s.ensureManagedDir(id, "assets"); err != nil {
		return err
	}
	return atomicBytes(filepath.Join(s.artifactDir(id), "assets", name), data, 0o600)
}

// GetAsset returns a stored asset while enforcing the configured read limit.
func (s *Store) GetAsset(id, name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := validateAssetName(name); err != nil {
		return nil, err
	}
	if err := s.ensureArtifactDir(id); err != nil {
		return nil, err
	}
	if err := s.ensureManagedDir(id, "assets"); err != nil {
		return nil, err
	}
	path := filepath.Join(s.artifactDir(id), "assets", name)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: asset %q", ErrNotFound, name)
		}
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > s.maxAsset {
		if info.Size() > s.maxAsset {
			return nil, ErrTooLarge
		}
		return nil, fmt.Errorf("%w: asset is not a regular file", ErrCorrupt)
	}
	return readLimited(path, s.maxAsset)
}

// Duplicate creates an independent artifact copy and copies its assets.
func (s *Store) Duplicate(sourceID, newID string) (*Artifact, error) {
	source, err := s.Get(sourceID)
	if err != nil {
		return nil, err
	}
	sourceRevision := source.Revision
	source.ID = newID
	source.Revision = 0
	source.CreatedAt = time.Time{}
	source.UpdatedAt = time.Time{}
	source.Provenance = append(source.Provenance, SourceReference{
		Kind: "artifact", ArtifactID: sourceID, Revision: sourceRevision,
	})
	created, err := s.Create(*source)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureManagedDir(sourceID, "assets"); err != nil {
		_ = os.RemoveAll(s.artifactDir(created.ID))
		return nil, err
	}
	if err := s.ensureManagedDir(created.ID, "assets"); err != nil {
		_ = os.RemoveAll(s.artifactDir(created.ID))
		return nil, err
	}
	sourceAssets := filepath.Join(s.artifactDir(sourceID), "assets")
	entries, err := os.ReadDir(sourceAssets)
	if err != nil {
		_ = os.RemoveAll(s.artifactDir(created.ID))
		return nil, fmt.Errorf("list source assets: %w", err)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || validateAssetName(entry.Name()) != nil {
			_ = os.RemoveAll(s.artifactDir(created.ID))
			return nil, fmt.Errorf("%w: unsafe source asset %q", ErrCorrupt, entry.Name())
		}
		data, err := readLimited(filepath.Join(sourceAssets, entry.Name()), s.maxAsset)
		if err != nil {
			_ = os.RemoveAll(s.artifactDir(created.ID))
			return nil, err
		}
		if err := atomicBytes(filepath.Join(s.artifactDir(created.ID), "assets", entry.Name()), data, 0o600); err != nil {
			_ = os.RemoveAll(s.artifactDir(created.ID))
			return nil, err
		}
	}
	return created, nil
}

func (s *Store) prepare(artifact *Artifact) error {
	if artifact.Renderer.ID == "" || artifact.Renderer.APIVersion == "" || artifact.Renderer.MediaType == "" {
		return errors.New("renderer ID, API version, and media type are required")
	}
	if len(artifact.Payload) == 0 || !json.Valid(artifact.Payload) {
		return errors.New("payload must be valid JSON")
	}
	size := int64(len(artifact.Payload))
	if artifact.Fallback != nil {
		if artifact.Fallback.MediaType == "" || len(artifact.Fallback.Data) == 0 || !json.Valid(artifact.Fallback.Data) {
			return errors.New("fallback requires a media type and valid JSON data")
		}
		size += int64(len(artifact.Fallback.Data))
	}
	if size > s.maxPayload {
		return ErrTooLarge
	}
	return nil
}

func (s *Store) getUnlocked(id string) (*Artifact, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := s.ensureArtifactDir(id); err != nil {
		return nil, err
	}
	var artifact Artifact
	if err := readJSON(filepath.Join(s.artifactDir(id), "artifact.json"), &artifact, s.documentLimit()); err != nil {
		return nil, mapReadError(err, id)
	}
	if artifact.ID != id || artifact.SchemaVersion <= 0 || artifact.SchemaVersion > CurrentSchemaVersion ||
		artifact.Revision == 0 || len(artifact.Payload) == 0 || !json.Valid(artifact.Payload) {
		return nil, fmt.Errorf("%w: invalid artifact %q", ErrCorrupt, id)
	}
	if artifact.Fallback != nil && !json.Valid(artifact.Fallback.Data) {
		return nil, fmt.Errorf("%w: invalid fallback in %q", ErrCorrupt, id)
	}
	return cloneArtifact(&artifact), nil
}

func (s *Store) writeRevision(artifact *Artifact) error {
	if err := s.ensureManagedDir(artifact.ID, "revisions"); err != nil {
		return err
	}
	snapshot := ArtifactRevision{
		ArtifactID: artifact.ID,
		Revision:   artifact.Revision,
		CreatedAt:  artifact.UpdatedAt,
		Artifact:   *cloneArtifact(artifact),
	}
	path := filepath.Join(s.artifactDir(artifact.ID), "revisions", fmt.Sprintf("%020d.json", artifact.Revision))
	return atomicJSON(path, &snapshot)
}

func (s *Store) artifactDir(id string) string { return filepath.Join(s.root, id) }

func (s *Store) ensureArtifactDir(id string) error {
	info, err := os.Lstat(s.artifactDir(id))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: artifact path is not a directory", ErrCorrupt)
	}
	return nil
}

func (s *Store) ensureManagedDir(id, name string) error {
	info, err := os.Lstat(filepath.Join(s.artifactDir(id), name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: missing %s directory", ErrCorrupt, name)
		}
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s path is not a directory", ErrCorrupt, name)
	}
	return nil
}

func (s *Store) documentLimit() int64 {
	if s.maxPayload > (1<<62)-(1<<20) {
		return 1 << 62
	}
	return s.maxPayload + (1 << 20)
}

func validateID(id string) error {
	if !safeID.MatchString(id) || id == "." || id == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return nil
}

func validateAssetName(name string) error {
	if !safeAsset.MatchString(name) || name == "." || name == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidAsset, name)
	}
	return nil
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("artifact-%d", time.Now().UnixNano())
}

func matches(a Artifact, f Filter) bool {
	if f.Kind != "" && a.Kind != f.Kind ||
		f.WorkspaceID != "" && a.Links.WorkspaceID != f.WorkspaceID ||
		f.ProjectID != "" && a.Links.ProjectID != f.ProjectID ||
		f.ChannelID != "" && a.Links.ChannelID != f.ChannelID ||
		f.CollaborationID != "" && a.Links.CollaborationID != f.CollaborationID ||
		f.RendererID != "" && a.Renderer.ID != f.RendererID {
		return false
	}
	if f.Capability != "" {
		for _, capability := range a.Capabilities {
			if capability == f.Capability {
				return true
			}
		}
		return false
	}
	return true
}

func cloneArtifact(a *Artifact) *Artifact {
	data, _ := json.Marshal(a)
	var clone Artifact
	_ = json.Unmarshal(data, &clone)
	return &clone
}

func readJSON(path string, target any, limit int64) error {
	data, err := readLimited(path, limit)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func readLimited(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrCorrupt, filepath.Base(path))
	}
	if info.Size() > limit {
		return nil, ErrTooLarge
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := io.LimitReader(file, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}

func mapReadError(err error, id string) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if errors.Is(err, ErrTooLarge) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrCorrupt, err)
}

func atomicJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	data = append(data, '\n')
	return atomicBytes(path, data, 0o600)
}

func atomicBytes(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return syncDir(dir)
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
