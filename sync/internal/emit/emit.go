// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package emit writes the protobuf text and puts it on disk.
//
// # One message per file
//
// A protobuf message cannot be continued across files, so the only
// decomposition available is one type at a time -- which keeps each file as
// small as the source model allows, and makes the file a stable address for a
// type rather than an accident of grouping.
//
// # Output matches buf format exactly
//
// The text written here is what `buf format` would produce, down to where a
// message literal collapses onto one line. That is deliberate: CI regenerates
// the tree and diffs it, and a generator whose raw output needed correcting
// would make that diff compare approximately rather than exactly.
package emit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/the-protobuf-project/vdm/sync/internal/model"
	"github.com/the-protobuf-project/vdm/sync/internal/plan"
	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// Emitter writes one model's packages.
type Emitter struct {
	M       *model.Model
	Planner *plan.Planner
}

// New returns an Emitter over m.
func New(m *model.Model) *Emitter {
	return &Emitter{M: m, Planner: plan.New(m)}
}

// Generate writes every package into out, replacing whatever is there.
//
// The tree is rewritten, not merged into. A type that stops being generated --
// a value object promoted to its own package, an enum replaced by a scalar --
// would otherwise leave its file behind, and a stale .proto still compiles and
// still lints, so nothing would report it.
func (e *Emitter) Generate(defs []sdl.Def, out string) (int, error) {
	if err := os.RemoveAll(out); err != nil {
		return 0, err
	}

	total, err := e.generateVocab(defs, out)
	if err != nil {
		return 0, err
	}
	for _, pkg := range e.M.Packages {
		n, err := e.generatePackage(pkg, out)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// generatePackage writes one package's type files and its method surface.
func (e *Emitter) generatePackage(pkg *model.Package, out string) (int, error) {
	files, err := e.planFiles(pkg)
	if err != nil {
		return 0, err
	}

	dir := filepath.Join(append([]string{out}, append(pkg.Segments(), "v1")...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}

	written := 0
	for _, f := range files {
		if err := writeFile(dir, f.Base, f.Body); err != nil {
			return 0, err
		}
		written++
	}
	// Rule 7: every resource gets its full standard method set. Which methods
	// depends on whether the parent holds one of this branch or many.
	for _, f := range []struct{ base, body string }{
		{"messages.proto", e.Messages(pkg)},
		{"service.proto", e.Service(pkg)},
	} {
		if err := writeFile(dir, f.base, f.body); err != nil {
			return 0, err
		}
		written++
	}
	return written, nil
}

// writeFile writes one generated file.
func writeFile(dir, base, body string) error {
	if err := os.WriteFile(filepath.Join(dir, base), []byte(body), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", base, err)
	}
	return nil
}
