package repo

import (
	"fmt"
	"testing"
)

func TestTrimRepositoryIndexFootprintSourceFiles(t *testing.T) {
	idx := &RepositoryIndex{Name: "test", SourceFiles: make(map[string]*SourceFile)}
	for i := 0; i < MaxIndexedSourceFilesInMemory+100; i++ {
		p := fmt.Sprintf("f%d", i)
		idx.SourceFiles[p] = &SourceFile{Path: p, Content: "x"}
	}
	TrimRepositoryIndexFootprint(idx)
	if len(idx.SourceFiles) != MaxIndexedSourceFilesInMemory {
		t.Fatalf("got %d files want %d", len(idx.SourceFiles), MaxIndexedSourceFilesInMemory)
	}
}

func TestTrimRepositoryIndexFootprintArchitecture(t *testing.T) {
	idx := &RepositoryIndex{Name: "x", SourceFiles: map[string]*SourceFile{}}
	idx.ArchitectureDoc = string(make([]byte, maxArchitectureDocBytes+1000))
	TrimRepositoryIndexFootprint(idx)
	if len(idx.ArchitectureDoc) > maxArchitectureDocBytes+200 {
		t.Fatalf("architecture doc still huge: %d", len(idx.ArchitectureDoc))
	}
}
