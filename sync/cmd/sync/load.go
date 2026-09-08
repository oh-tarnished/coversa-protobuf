// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// load.go reads the specification tree off disk.

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// loadTree parses every .graphql file under root, in a stable order.
//
// Sorted because the output must not depend on directory iteration order: two
// runs over the same checkout have to produce byte-identical files, or CI's
// regenerate-and-diff reports a change that nobody made.
func loadTree(root string) ([]sdl.Def, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".graphql") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var defs []sdl.Def
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		parsed, err := sdl.Parse(f, string(src))
		if err != nil {
			return nil, err
		}
		defs = append(defs, parsed...)
	}
	return defs, nil
}
