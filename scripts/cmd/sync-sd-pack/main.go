// Command sync-sd-pack copies a local software-development pack folder into
// ~/.neural-junkie/packs and records packs.dev_sources (dev-link).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/camronwood/neural-junkie/internal/config"
)

func main() {
	packDir := flag.String("pack-dir", "", "path to neural-junkie-pack-software-development")
	flag.Parse()
	if *packDir == "" {
		fmt.Fprintln(os.Stderr, "--pack-dir is required")
		os.Exit(2)
	}
	abs, err := filepath.Abs(*packDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pack-dir: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	m, err := cfg.DevLinkPack(abs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dev-link: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.SetPackEnabled(m.ID, true); err != nil {
		fmt.Fprintf(os.Stderr, "enable pack: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("dev-linked %s@%s → ~/.neural-junkie/packs/%s (enabled)\n", m.ID, m.Version, m.ID)
}
