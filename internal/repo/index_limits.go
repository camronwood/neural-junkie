package repo

import (
	"log"
	"sort"
)

// MaxIndexedSourceFilesInMemory caps how many source file bodies we keep per
// RepositoryIndex in RAM (and on disk after next save). Each entry can be up
// to ~MaxFileSize compressed; without a cap, large monorepos can exceed 10GB+.
const MaxIndexedSourceFilesInMemory = 2000

const maxArchitectureDocBytes = 512 * 1024

const maxKeyFileContentBytes = 512 * 1024

// TrimRepositoryIndexFootprint shrinks an index after load (or before save) so
// the hub/repo agent process does not retain unbounded compressed file payloads.
func TrimRepositoryIndexFootprint(idx *RepositoryIndex) {
	if idx == nil {
		return
	}
	if len(idx.SourceFiles) <= MaxIndexedSourceFilesInMemory {
		trimLargeStrings(idx)
		return
	}
	paths := make([]string, 0, len(idx.SourceFiles))
	for p := range idx.SourceFiles {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	drop := paths[:len(paths)-MaxIndexedSourceFilesInMemory]
	for _, p := range drop {
		delete(idx.SourceFiles, p)
	}
	log.Printf("[repo] Trimmed repository index %q: kept %d source file bodies (cap=%d)",
		idx.Name, len(idx.SourceFiles), MaxIndexedSourceFilesInMemory)
	trimLargeStrings(idx)
}

func trimLargeStrings(idx *RepositoryIndex) {
	if len(idx.ArchitectureDoc) > maxArchitectureDocBytes {
		idx.ArchitectureDoc = idx.ArchitectureDoc[:maxArchitectureDocBytes] + "\n\n*[Architecture doc truncated for memory cap]*\n"
	}
	for k, v := range idx.KeyFiles {
		if len(v) > maxKeyFileContentBytes {
			idx.KeyFiles[k] = v[:maxKeyFileContentBytes] + "\n\n*[truncated]*\n"
		}
	}
}
