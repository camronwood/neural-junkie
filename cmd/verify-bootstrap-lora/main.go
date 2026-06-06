package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hfhub"
)

func main() {
	base := flag.String("base", "", "Ollama base model tag")
	adapter := flag.String("adapter", "", "Local adapter directory")
	tag := flag.String("tag", "", "Composed Ollama tag")
	flag.Parse()

	if strings.TrimSpace(*base) == "" || strings.TrimSpace(*adapter) == "" || strings.TrimSpace(*tag) == "" {
		fmt.Fprintln(os.Stderr, "usage: verify-bootstrap-lora -base TAG -adapter DIR -tag TAG")
		os.Exit(2)
	}

	if msg := hfhub.WarnAdapterBaseMismatch(*base, *adapter); msg != "" {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", msg)
	}

	ctx := context.Background()
	if err := hfhub.ImportAdapterToOllama(ctx, *base, *adapter, *tag); err != nil {
		fmt.Fprintf(os.Stderr, "compose failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("composed %q on base %q\n", *tag, *base)
}
