package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/git"
	"github.com/camronwood/neural-junkie/internal/workspacefiles"
	"github.com/camronwood/neural-junkie/internal/workspacesymbols"
)

const (
	maxFilesPerBuild = 8000
	maxUINodes       = 400
	maxUIEdges       = 800
	godNodeLimit     = 24
)

var (
	buildMu  sync.Mutex
	building = map[string]bool{}
)

// RepoHash returns a stable hash for a repo path.
func RepoHash(repoPath string) string {
	abs, _ := filepath.Abs(filepath.Clean(repoPath))
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:8])
}

func graphDir(repoPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "code-graph", RepoHash(repoPath)), nil
}

// Status returns graph build metadata.
func Status(repoPath string) (Meta, error) {
	meta := Meta{RepoPath: repoPath, RepoHash: RepoHash(repoPath)}
	dir, err := graphDir(repoPath)
	if err != nil {
		return meta, err
	}
	store, err := Open(dir)
	if err != nil {
		return meta, err
	}
	defer store.Close()
	loaded, err := store.LoadMeta()
	if err != nil {
		return meta, err
	}
	if loaded.RepoHash != "" {
		meta = loaded
	}
	buildMu.Lock()
	meta.Building = building[RepoHash(repoPath)]
	buildMu.Unlock()
	return meta, nil
}

// BuildIndexAsync starts a background graph build.
func BuildIndexAsync(repoPath string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_ = BuildIndex(ctx, repoPath)
	}()
}

// RebuildIndexAsync deletes the cached graph and rebuilds.
func RebuildIndexAsync(repoPath string) {
	if dir, err := graphDir(repoPath); err == nil {
		_ = os.RemoveAll(dir)
	}
	BuildIndexAsync(repoPath)
}

// BuildIndex constructs the knowledge graph for a repository.
func BuildIndex(ctx context.Context, repoPath string) error {
	repoPath = filepath.Clean(repoPath)
	hash := RepoHash(repoPath)
	buildMu.Lock()
	if building[hash] {
		buildMu.Unlock()
		return nil
	}
	building[hash] = true
	buildMu.Unlock()
	defer func() {
		buildMu.Lock()
		delete(building, hash)
		buildMu.Unlock()
	}()

	dir, err := graphDir(repoPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	head := ""
	if git.IsRepo(repoPath) {
		head, _ = git.RevParseHEAD(repoPath)
	}

	store, err := Open(dir)
	if err != nil {
		return err
	}
	defer store.Close()

	prev, _ := store.LoadMeta()
	if prev.Ready && prev.GitHEAD == head && prev.GitHEAD != "" && prev.NodeCount > 0 {
		return nil
	}

	files, err := workspacefiles.Search(ctx, repoPath, "", maxFilesPerBuild)
	if err != nil {
		return err
	}

	symbols, _ := workspacesymbols.BuildIndex(ctx, repoPath)

	repoLabel := filepath.Base(repoPath)
	repoID := "repo:" + hash
	nodesByID := map[string]*Node{
		repoID: {ID: repoID, Kind: NodeRepo, Label: repoLabel, Community: "root"},
	}
	var edges []Edge
	fileIDs := map[string]string{}
	packageIDs := map[string]string{}

	addEdge := func(from, to string, kind EdgeKind, prov Provenance, file string, line int) {
		if from == "" || to == "" || from == to {
			return
		}
		id := edgeID(from, to, kind, line)
		edges = append(edges, Edge{
			ID: id, From: from, To: to, Kind: kind, Provenance: prov, Path: file, Line: line,
		})
	}

	for _, rel := range files {
		rel = filepath.ToSlash(rel)
		ext := strings.ToLower(filepath.Ext(rel))
		switch ext {
		case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".md", ".toml", ".yaml", ".yml", ".json":
		default:
			continue
		}
		comm := packageCommunity(rel)
		pkgID := "pkg:" + comm
		if _, ok := packageIDs[comm]; !ok {
			packageIDs[comm] = pkgID
			nodesByID[pkgID] = &Node{
				ID: pkgID, Kind: NodePackage, Label: comm, Community: comm, Path: path.Dir(rel),
			}
			addEdge(repoID, pkgID, EdgeContains, ProvenanceExtracted, "", 0)
		}
		fileID := "file:" + rel
		fileIDs[rel] = fileID
		nodesByID[fileID] = &Node{
			ID: fileID, Kind: NodeFile, Label: path.Base(rel), Path: rel, Community: comm,
			Language: langForExt(ext),
		}
		addEdge(pkgID, fileID, EdgeContains, ProvenanceExtracted, rel, 0)
	}

	for _, sym := range symbols {
		rel := filepath.ToSlash(sym.Path)
		fileID, ok := fileIDs[rel]
		if !ok {
			continue
		}
		comm := packageCommunity(rel)
		symID := fmt.Sprintf("sym:%s:%d:%s", rel, sym.Line, sym.Name)
		nodesByID[symID] = &Node{
			ID: symID, Kind: NodeSymbol, Label: sym.Name, Path: rel, Line: sym.Line,
			Language: sym.Language, Community: comm, SymbolKind: sym.Kind,
		}
		addEdge(fileID, symID, EdgeDefines, ProvenanceExtracted, rel, sym.Line)
		addEdge(fileID, symID, EdgeContains, ProvenanceExtracted, rel, sym.Line)
	}

	var moduleRoots []string
	if modBytes, err := os.ReadFile(filepath.Join(repoPath, "go.mod")); err == nil {
		if mp := readGoModulePath(string(modBytes)); mp != "" {
			moduleRoots = append(moduleRoots, mp)
		}
	}

	for rel, fileID := range fileIDs {
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		imps := extractImports(rel, string(b))
		for _, im := range imps {
			// External/unresolved import target as a lightweight package-like node for density.
			importNodeID := "import:" + im.Target
			if _, ok := nodesByID[importNodeID]; !ok {
				nodesByID[importNodeID] = &Node{
					ID: importNodeID, Kind: NodePackage, Label: im.Target,
					Community: packageCommunity(rel), Path: im.Target,
				}
			}
			addEdge(fileID, importNodeID, EdgeImports, ProvenanceExtracted, rel, im.Line)
			if targetID, ok := resolveImportTarget(rel, im.Target, fileIDs, moduleRoots); ok {
				addEdge(fileID, targetID, EdgeResolvesTo, ProvenanceInferred, rel, im.Line)
				addEdge(fileID, targetID, EdgeImports, ProvenanceInferred, rel, im.Line)
			}
		}
	}

	// Degree + community colors
	degree := map[string]int{}
	for _, e := range edges {
		degree[e.From]++
		degree[e.To]++
	}
	nodes := make([]Node, 0, len(nodesByID))
	for _, n := range nodesByID {
		n.Degree = degree[n.ID]
		nodes = append(nodes, *n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	meta := Meta{
		RepoPath:    repoPath,
		RepoHash:    hash,
		NodeCount:   len(nodes),
		EdgeCount:   len(edges),
		GitHEAD:     head,
		LastBuiltAt: time.Now().UTC(),
		Ready:       len(nodes) > 0,
	}
	if err := store.ReplaceAll(nodes, edges, meta); err != nil {
		return err
	}
	_ = writeReport(dir, nodes, edges, meta)
	return nil
}

func langForExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

// EnsureReady builds the graph if missing.
func EnsureReady(ctx context.Context, repoPath string) (Meta, error) {
	meta, err := Status(repoPath)
	if err != nil {
		return meta, err
	}
	if meta.Ready {
		return meta, nil
	}
	if !meta.Building {
		BuildIndexAsync(repoPath)
	}
	return Status(repoPath)
}

// OpenStore opens the store for a repo.
func OpenStore(repoPath string) (*Store, error) {
	dir, err := graphDir(repoPath)
	if err != nil {
		return nil, err
	}
	return Open(dir)
}
