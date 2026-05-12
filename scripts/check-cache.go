package main

import (
	"fmt"
	"os"

	"github.com/camronwood/neural-junkie/internal/repo"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run check-cache.go <repo-path>")
		fmt.Println("Example: go run check-cache.go /Users/camronwood/development/sandbox/neural-junkie")
		os.Exit(1)
	}

	repoPath := os.Args[1]

	storage, err := repo.NewStorage()
	if err != nil {
		fmt.Printf("❌ Error creating storage: %v\n", err)
		os.Exit(1)
	}

	cacheKey, err := storage.GetCacheKeyForPath(repoPath)
	if err != nil {
		fmt.Printf("❌ Error generating cache key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📂 Repository Path: %s\n", repoPath)
	fmt.Printf("🔑 Cache Key: %s\n", cacheKey)
	fmt.Printf("📁 Cache Location: ~/.neural-junkie/repos/%s/\n", cacheKey)
	fmt.Println()

	if storage.IndexExists(cacheKey) {
		fmt.Println("✅ Cache EXISTS - will load instantly!")

		// Try to load metadata
		metadata, err := storage.LoadMetadata(cacheKey)
		if err == nil {
			fmt.Printf("📋 Agents that have used this cache: %v\n", metadata.AgentNames)
			fmt.Printf("🗂️  Repository: %s\n", metadata.Path)
		}

		// Try to load index
		index, err := storage.LoadIndex(cacheKey)
		if err == nil {
			fmt.Printf("\n📊 Index Details:\n")
			fmt.Printf("   - Files: %d\n", index.FileCount)
			fmt.Printf("   - Total Size: %.2f MB\n", float64(index.TotalSize)/(1024*1024))
			fmt.Printf("   - Last Indexed: %s\n", index.LastIndexed.Format("2006-01-02 15:04:05"))
			fmt.Printf("   - Code Patterns: %v\n", index.CodePatterns)
		}
	} else {
		fmt.Println("❌ Cache DOES NOT EXIST - will perform full indexing")
		fmt.Println("💡 First indexing may take 30-60 seconds")
		fmt.Println("💡 Future agents for this repo will load instantly!")
	}
}
