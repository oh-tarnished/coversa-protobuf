// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package load reads both specifications and builds the model.
//
// Every command does the same three things — read the pins, read what they
// point at, partition it — and each did them slightly differently while the
// code was duplicated across four `main` packages. It is one function here so
// the documentation, the schema and the mapping manifest cannot be generated
// from three subtly different models.
package load

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/sdl"
	"github.com/the-protobuf-project/vdm/sync/spec"
	"github.com/the-protobuf-project/vdm/sync/vdm"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// Model reads the pins at path and returns the model they describe.
func Model(path string) (*model.Model, error) {
	s, err := spec.Load(path)
	if err != nil {
		return nil, err
	}

	vehicle, err := vspec.Parse(s.VSS.Path)
	if err != nil {
		return nil, fmt.Errorf("vss: %w", err)
	}
	units, err := vspec.LoadCatalogue(filepath.Dir(s.VSS.Path))
	if err != nil {
		return nil, fmt.Errorf("vss: %w", err)
	}

	defs, err := parseSDL(s.VDM.Path)
	if err != nil {
		return nil, fmt.Errorf("vdm: %w", err)
	}
	others, err := vdm.Convert(defs)
	if err != nil {
		return nil, fmt.Errorf("vdm: %w", err)
	}

	m, err := model.Build(vehicle, others, units)
	if err != nil {
		return nil, err
	}
	m.Spec = s

	// Both specifications are in hand, so a resolution table entry naming no
	// signal is now answerable -- and is a stale decision rather than a
	// missing one. See model.CheckCatalogue.
	if err := m.CheckCatalogue(); err != nil {
		return nil, err
	}
	return m, nil
}

// parseSDL reads every .graphql file under root, in a stable order.
//
// Sorted because the output must not depend on directory iteration order: two
// runs over the same checkout have to produce byte-identical files, or CI's
// regenerate-and-diff reports a change nobody made.
func parseSDL(root string) ([]sdl.Def, error) {
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
