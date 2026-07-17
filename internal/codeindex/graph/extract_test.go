package graph

import (
	"strings"
	"testing"
)

func TestExtractGoImports(t *testing.T) {
	src := `package main

import (
	"fmt"
	"github.com/foo/bar"
)

import "os"
`
	imps := extractGoImports(src)
	if len(imps) < 3 {
		t.Fatalf("expected >=3 imports, got %d: %+v", len(imps), imps)
	}
	joined := ""
	for _, im := range imps {
		joined += im.Target + " "
	}
	for _, want := range []string{"fmt", "github.com/foo/bar", "os"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing import %q in %q", want, joined)
		}
	}
}

func TestExtractTSImports(t *testing.T) {
	src := `
import React from 'react';
import { x } from './utils';
const y = require('../lib');
`
	imps := extractTSImports(src)
	if len(imps) < 3 {
		t.Fatalf("expected >=3 imports, got %d", len(imps))
	}
}

func TestPackageCommunity(t *testing.T) {
	if got := packageCommunity("internal/hub/commands.go"); got != "internal/hub" {
		t.Fatalf("got %q", got)
	}
	if got := packageCommunity("main.go"); got != "root" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveTSImport(t *testing.T) {
	files := map[string]string{
		"src/utils.ts": "file:src/utils.ts",
		"src/app.ts":   "file:src/app.ts",
	}
	id, ok := resolveImportTarget("src/app.ts", "./utils", files, nil)
	if !ok || id != "file:src/utils.ts" {
		t.Fatalf("got %q ok=%v", id, ok)
	}
}
